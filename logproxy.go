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
	Source       string
	Sink         string
	Format       string
	Tags         []Tag
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

	sourceUrl := GetIpfsLogAddress(lp.Source, "json")
	resp, err := http.Get(sourceUrl)
	if err != nil {
		errlog.Println("Get log stream: ", err)
		panic(err)
	}
	lp.sourceStream = resp.Body

	go lp.ReadSource()
	go lp.FilterEvents()
	go lp.WriteSink()
	//List use to keep track of active collections
	proxyList[lp.Name] = lp
}

//Read from the source -> Filter
func (lp *LogProxy) ReadSource() {
	infolog.Printf("Reader Open In-Stream: %s\n", lp.Source)
	dec := json.NewDecoder(lp.sourceStream)
	for {
		select {
		case <-lp.ctx.Done():
			infolog.Printf("Reader Close In-Stream: %s\n", lp.Source)
			lp.sourceStream.Close()
			return
		default:
			var event LogEvent
			if err := dec.Decode(&event.Message); err != nil {
				errlog.Println("Read Source Decode: ", err)
				return
			}
			lp.Inbound <- event
		}
	}

}

//Apply filters to event
func (lp *LogProxy) FilterEvents() {
	infolog.Printf("Filter Open In-Stream: %s\n", lp.Source)
	for {
		select {
		case <-lp.ctx.Done():
			infolog.Printf("Filter Close In-Stream: %s\n", lp.Source)
			return
		case event := <-lp.Inbound:
			//for _, filter := range lp.Filters {
			//	event = filter(event)
			//}
			//For Now
			event.AddTags(lp.Tags)
			lp.Outbound <- event
		}
	}
}

//Write log events to sink
func (lp *LogProxy) WriteSink() {
	infolog.Printf("Writer Open Out-Stream: %s\n", lp.Sink)
	for {
		select {
		case event := <-lp.Outbound:
			var b []byte
			var err error
			if lp.Format == "lineprotocol" {
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
			if lp.Sink == "stdout" {
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
			infolog.Printf("Writer Close Out-Stream: %s\n", lp.Sink)
			return
		}
	}
}

func (lp *LogProxy) Close() {
	infolog.Printf("\nClosing Connection: %s\n", lp.Name)
	lp.cancel()
}
