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
	"strings"
	"time"
)

type LogProxy struct {
	Name         string
	Source       string
	Sink         string
	Format       string
	Tags         []string
	sourceStream io.ReadCloser
	sinkStream   io.WriteCloser
	Inbound      chan LogEvent
	Outbound     chan LogEvent
	ctx          context.Context
	cancel       func()
	Filters      []func(LogEvent) LogEvent
}

type LogEvent struct {
	Message map[string]interface{}
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

//Apply filters to event
func (lp *LogProxy) FilterEvents() {
	infolog.Printf("Filter Open In-Stream: %s\n", lp.Source)
	for {
		select {
		case <-lp.ctx.Done():
			infolog.Printf("Filter Close In-Stream: %s\n", lp.Source)
			return
		case event := <-lp.Inbound:
			for _, filter := range lp.Filters {
				event = filter(event)
			}
			lp.Outbound <- event
		}
	}
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

//Write log events to sink
func (lp *LogProxy) WriteSink() {
	infolog.Printf("Writer Open Out-Stream: %s\n", lp.Sink)
	for {
		select {
		case event := <-lp.Outbound:
			var b []byte
			var err error
			if lp.Format == "lineProtocol" {
				b = lp.LineProtocol(event)
			} else {
				b, err = json.Marshal(event.Message)
				if err != nil {
					errlog.Printf("Write Sink marshal: %#v", event)
					continue
				}
			}
			if lp.Sink == "stdout" {
				os.Stdout.Write([]byte(b))
			} else {
				url := fmt.Sprintf("http://%s/write?db=mydb", lp.Sink)
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

//Line protocol is the prefered method for writing to influxdb
//https://docs.influxdata.com/influxdb/v1.3/write_protocols/line_protocol_tutorial/
//measure, tag1=value1,...,tagn=valuen field1=value1,...,fieldn=valuen time (unixnano)
//Example:
//dht,nodeId=QmcJ9RHiEoa1WYeaFAEHVgjc41aXfD52WDEFLZrEcQvbPR,event=findPeerSingleBegin duration=0 1510956550223924627
//swarm2,nodeId=QmcJ9RHiEoa1WYeaFAEHVgjc41aXfD52WDEFLZrEcQvbPR,event=swarmDialAttemptSync duration=1129297969.000000 1510956550080102777
func (lp *LogProxy) LineProtocol(event LogEvent) []byte {
	var tags []string
	var fields []string

	//Duration is a field, and line protocol must have at least 1 field
	if event.Message["duration"] == nil {
		event.Message["duration"] = 0
	}

	tags = lp.Tags
	knownFields := []string{"duration"}
	measurement := event.Message["system"]
	knownTags := []string{"session", "subsystem", "event", "requestId"}

	//Could panic if tag is not a string..
	for _, tag := range knownTags {
		if event.Message[tag] != nil && len(event.Message[tag].(string)) != 0 {
			value := fmt.Sprintf("%s=%s", tag, event.Message[tag].(string))
			tags = append(tags, value)
		}
	}
	for _, field := range knownFields {
		if event.Message[field] == nil {
			continue
		}
		value := fmt.Sprintf("%s=%s", field, stringifyInterface(event.Message[field]))
		fields = append(fields, value)
	}

	parsedTime, err := time.Parse(time.RFC3339Nano, event.Message["time"].(string))
	if err != nil {
		panic(err)
	}

	if len(tags) == 0 {
		return []byte(fmt.Sprintf("%s %s %d\n", measurement, strings.Join(fields, ","), parsedTime.UnixNano()))
	} else {
		return []byte(fmt.Sprintf("%s,%s %s %d\n", measurement, strings.Join(tags, ","), strings.Join(fields, ","), parsedTime.UnixNano()))
	}
}

//Bare minimum, needs work
//TODO add more cases as they arrise
func stringifyInterface(e interface{}) string {
	switch e.(type) {
	case uint, uint8, uint16, uint32, uint64, int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", e)
	case float64:
		return fmt.Sprintf("%f", e)
	default:
		errlog.Fatalf("Unknown Type: %#v", e)
		panic(e)
	}
}
