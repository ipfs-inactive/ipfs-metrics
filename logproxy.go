package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
)

type LogProxy struct {
	Name         string
	Source       Source
	Sink         Sink
	sourceStream io.ReadCloser
	sinkStream   io.WriteCloser
	Inbound      chan LogEvent
	Outbound     chan LogEvent
	ctx          context.Context
	cancel       func()
	Filters      []func(LogEvent) LogEvent
}

//Start a log proxy
func (lp *LogProxy) Start() {
	lp.ctx, lp.cancel = context.WithCancel(context.Background())

	sourceUrl := GetIpfsLogAddress(lp.Source)
	resp, err := http.Get(sourceUrl)
	if err != nil {
		errlog.Println("Get log stream: ", err)
		return
	}
	lp.sourceStream = resp.Body

	//If we are not writing to stdout, and the format is lineprotocol
	//we are probably writing to influxdb, so ensure the db exists
	if len(lp.Sink.Address) != 0 && lp.Sink.Format == "lineprotocol" {
		_, err := CreateDatabase(db, lp.Sink)
		if err != nil {
			errlog.Println("Failed to create database: ", err)
			panic("Please ensure that influxdb is running")
		}
		infolog.Print("database found!")
	}

	infolog.Printf("Opening Connection Name: %s\n", lp.Name)
	go lp.ReadSource()
	go lp.FilterEvents()
	go lp.WriteSink()
	//List use to keep track of active collections
	proxyList[lp.Name] = lp
}

//Read from the source -> Filter
func (lp *LogProxy) ReadSource() {
	infolog.Printf("Reader Open In-Stream: %s Name: %s\n", lp.Source, lp.Name)
	dec := json.NewDecoder(lp.sourceStream)
	for {
		select {
		case <-lp.ctx.Done():
			infolog.Printf("Reader Close In-Stream: %s Name: %s\n", lp.Source, lp.Name)
			lp.sourceStream.Close()
			return
		default:
			var event LogEvent
			if err := dec.Decode(&event.Message); err != nil {
				errlog.Printf("Read Source: %s decode error: %v", lp.Source, err)
				return
			}
			lp.Inbound <- event
		}
	}

}

//Apply filters to event
func (lp *LogProxy) FilterEvents() {
	infolog.Printf("Filter Open In-Stream: %s Name: %s\n", lp.Source, lp.Name)
	for {
		select {
		case <-lp.ctx.Done():
			infolog.Printf("Filter Close In-Stream: %s Name: %s\n", lp.Source, lp.Name)
			return
		case event := <-lp.Inbound:
			//for _, filter := range lp.Filters {
			//	event = filter(event)
			//}
			event.AddTags(lp.Source.Tags)
			lp.Outbound <- event
		}
	}
}

//Write log events to sink
func (lp *LogProxy) WriteSink() {
	infolog.Printf("Writer Open Out-Stream: %s Name: %s\n", lp.Sink, lp.Name)
	for {
		select {
		case event := <-lp.Outbound:
			var b []byte
			var err error
			if lp.Sink.Format == "lineprotocol" {
				b, err = event.ToLP()
				if err != nil {
					errlog.Println("Write Sink marshal: ", err)
					continue
				}
			} else {
				b, err = event.ToJSON()
				if err != nil {
					errlog.Println("Write Sink marshal: ", err)
					continue
				}
			}
			if len(lp.Sink.Address) == 0 {
				os.Stdout.Write(b)
			} else {
				url := fmt.Sprintf("http://%s/write?db=%s", lp.Sink, db)
				resp, err := http.Post(url, "application/octet-stream", bytes.NewBuffer(b))
				if err != nil {
					errlog.Printf("Did you forget to include the port? Inlfux is usualy on 8086")
					panic(err)
				}
				defer resp.Body.Close()
				if resp.StatusCode != 204 {
					errlog.Printf("Write Sink write: %#v", event)
					errlog.Println("Status: ", resp.Status)
					errlog.Println("Headers: ", resp.Header)
					body, _ := ioutil.ReadAll(resp.Body)
					errlog.Println("Body: ", string(body))
				}

			}

		case <-lp.ctx.Done():
			infolog.Printf("Writer Close Out-Stream: %s Name: %s\n", lp.Sink, lp.Name)
			return
		}
	}
}

func (lp *LogProxy) Close() {
	infolog.Printf("Closing Connection Name: %s\n", lp.Name)
	lp.cancel()
}
