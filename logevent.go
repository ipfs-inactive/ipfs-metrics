package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var messageFields = []string{"duration"}
var messageTags = []string{"session", "subsystem", "event", "requestId"}

type LogEvent struct {
	Message map[string]interface{} `json:"message"`
	Tags    []Tag                  `json:"tags"`
}

func (le *LogEvent) AddTags(tags []Tag) {
	le.Tags = append(le.Tags, tags...)
}

func (le *LogEvent) AddTag(tag Tag) {
	le.Tags = append(le.Tags, tag)
}

func (le *LogEvent) ToJSON() ([]byte, error) {
	var b []byte
	//TODO: Replace in go-log
	if le.Message["duration"] == nil {
		le.Message["duration"] = 0
	}
	b, err := json.Marshal(le)
	if err != nil {
		return nil, err
	}
	return b, nil
}

//Line protocol is the prefered method for writing to influxdb
//https://docs.influxdata.com/influxdb/v1.3/write_protocols/line_protocol_tutorial/
//measure, tag1=value1,...,tagn=valuen field1=value1,...,fieldn=valuen time (unixnano)
//Example:
//dht,nodeId=QmcJ9RHiEoa1WYeaFAEHVgjc41aXfD52WDEFLZrEcQvbPR,event=findPeerSingleBegin duration=0 1510956550223924627
//swarm2,nodeId=QmcJ9RHiEoa1WYeaFAEHVgjc41aXfD52WDEFLZrEcQvbPR,event=swarmDialAttemptSync duration=1129297969.000000 1510956550080102777
//Returns a log event in Line Protocol Format
func (le *LogEvent) ToLP() ([]byte, error) {
	//Duration is a field, and line protocol must have at least 1 field
	//TODO: Replace in go-log
	if le.Message["duration"] == nil {
		le.Message["duration"] = 0
	}

	measurement := le.Message["system"]
	fields, err := le.getLPFields()
	if err != nil {
		return nil, err
	}
	tags, err := le.getLPTags()
	if err != nil {
		return nil, err
	}
	ts, err := le.getLPTime()
	if err != nil {
		return nil, err
	}

	if len(tags) == 0 {
		return []byte(fmt.Sprintf("%s %s %d\n", measurement, strings.Join(fields, ","), ts)), nil
	} else {
		return []byte(fmt.Sprintf("%s,%s %s %d\n", measurement, strings.Join(tags, ","), strings.Join(fields, ","), ts)), nil
	}

}

//Returns array of event tags in line protocol format
func (le *LogEvent) getLPTags() ([]string, error) {
	var tags []string
	for _, tag := range messageTags {
		//if the messages contains the tag
		if le.Message[tag] != nil && len(le.Message[tag].(string)) != 0 {
			value := fmt.Sprintf("%s=%s", tag, le.Message[tag].(string))
			tags = append(tags, value)
		}
	}
	for t := range le.Tags {
		value := fmt.Sprintf("%s=%s", le.Tags[t].Name, le.Tags[t].Value)
		tags = append(tags, value)
	}
	return tags, nil
}

func (le *LogEvent) getLPFields() ([]string, error) {
	var fields []string
	for _, field := range messageFields {
		if le.Message[field] == nil {
			continue
		}
		si, err := stringifyInterface(le.Message[field])
		if err != nil {
			return nil, err
		}
		value := fmt.Sprintf("%s=%s", field, si)
		fields = append(fields, value)
	}
	return fields, nil
}

func (le *LogEvent) getLPTime() (int64, error) {
	t, err := time.Parse(time.RFC3339Nano, le.Message["time"].(string))
	if err != nil {
		return -1, err
	}
	return t.UnixNano(), nil
}

//Bare minimum, needs work
//TODO add more cases as they arrise
func stringifyInterface(e interface{}) (string, error) {
	switch e.(type) {
	case uint, uint8, uint16, uint32, uint64, int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", e), nil
	case float64:
		return fmt.Sprintf("%f", e), nil
	default:
		errlog.Fatalf("Unknown Type: %#v", e)
		return "", errors.New(fmt.Sprintf("Unknown Type: %#v", e))
	}
}
