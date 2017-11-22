package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	cli "github.com/codegangsta/cli"
)

//return nil if valid, error if else
func ValidConfig(config *Config) error {
	if len(config.Source.Address) == 0 || len(config.Source.Port) == 0 {
		return errors.New("Invalid config, no source specified")
	}
	if len(config.Sink.Address) != 0 && len(config.Sink.Port) == 0 {
		return errors.New("invalid config, no sink port given")
	}
	if len(config.Sink.Address) == 0 && len(config.Sink.Port) != 0 {
		return errors.New("invalid config, no sink address given")
	}
	if len(config.Sink.Format) == 0 {
		return errors.New("invalid config, no sink format given")
	}
	format := strings.ToLower(config.Sink.Format)
	if !(format == "json" || format == "lineprotocol") {
		return errors.New(fmt.Sprintf("invalid config, unknown format: %s", format))
	}
	return nil
}

//returns a command or erros if invlaid options given
func NewAddCommand(c *cli.Context) (*Command, error) {
	var cmd Command
	cmd.Type = "add"
	if len(c.String("config")) != 0 {
		config, err := LoadConfig(c.String("config"))
		if err != nil {
			return nil, err
		}
		//ensure we are pased valid config options
		if err := ValidConfig(config); err != nil {
			return nil, err
		}
		cmd.Source = fmt.Sprintf("%s:%s", config.Source.Address, config.Source.Port)
		cmd.Tags = config.Source.Tags
		cmd.Format = strings.ToLower(config.Sink.Format)

		if len(config.Sink.Address) == 0 {
			infolog.Println("No output given, will write to stdout")
			cmd.Sink = "stdout"
		} else {
			cmd.Sink = fmt.Sprintf("%s:%s", config.Sink.Address, config.Sink.Port)
		}
	} else {
		//TODO's(forrestweston):
		// add -a (address) -p (port) options for cli input
		// add format field (to replace --lineprotocol field)
		// add enums for things like "output" and format
		if len(c.String("input")) == 0 {
			return nil, errors.New("Input of event logs required")
		} else {
			cmd.Source = c.String("input")
		}

		if len(c.String("output")) == 0 {
			infolog.Println("No output given, will write to stdout")
			cmd.Sink = "stdout"
		} else {
			cmd.Sink = c.String("output")
		}

		if c.Bool("lineprotocol") {
			cmd.Format = "lineprotocol"
		} else {
			cmd.Format = "json"
		}
	}
	//A default Tag, a liveness check for ipfs daemon, and a name for the collection when removing/listing
	nodeId, err := GetNodeId(cmd.Source)
	if err != nil {
		return nil, err
	}
	cmd.Tags = append(cmd.Tags, MakeTag("nodeId", nodeId))
	ts, err := MakeTags(c.Args())
	if err != nil {
		return nil, err
	}
	cmd.Tags = append(cmd.Tags, ts...)
	cmd.Node = nodeId
	return &cmd, nil
}

func CreateDatabase(dbName string) (*http.Response, error) {
	influxUrl := "http://localhost:8086"
	resource := "/query"
	data := url.Values{}
	data.Set("q", fmt.Sprintf("CREATE DATABASE %s", dbName))

	u, _ := url.ParseRequestURI(influxUrl)
	u.Path = resource
	urlStr := u.String()

	client := &http.Client{}
	r, err := http.NewRequest("POST", urlStr, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

	resp, err := client.Do(r)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

func SendCommand(c *Command) (*http.Response, error) {
	b, err := json.Marshal(c)
	if err != nil {
		errlog.Fatal("Failed to Marshal Command: ", err)
		return nil, err
	}

	url := fmt.Sprintf("http://localhost%s", port)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(b))
	if err != nil {
		errlog.Println("Failed to Create Request: ", err)
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		errlog.Println(err)
		return nil, err
	}
	return resp, nil
}

func GetIpfsLogAddress(multiadder, encoding string) string {
	return fmt.Sprintf("http://%s/api/v0/log/tail?encoding=%s&stream-channels=true", multiadder, encoding)
}

func GetNodeId(multiadder string) (string, error) {
	url := fmt.Sprintf("http://%s/api/v0/id", multiadder)
	resp, err := http.Get(url)
	if err != nil {
		errlog.Printf("Get NodeId, is the ipfs daemon running?\n")
		return "", err
	}
	var nodeInfo map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&nodeInfo)
	if err != nil {
		errlog.Println(err)
		return "", err
	}
	nodeId := nodeInfo["ID"].(string)
	if nodeId == "" {
		return "", errors.New("Could not get NodeID, are you sure this is an ipfs daemon?")
	}
	return nodeId, nil
}
