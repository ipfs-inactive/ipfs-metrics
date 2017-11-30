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

//returns a command or erros if invlaid options given
func NewAddCommand(c *cli.Context) (*Command, error) {
	var err error
	config := new(Config)
	//we will load file config, or use
	if len(c.String("config")) != 0 {
		config, err = LoadConfigFromFile(c.String("config"))
	} else {
		config, err = LoadConfigFromArgs(c)
	}
	if err != nil {
		return nil, err
	}
	//ensure we are pased valid config options
	if err := ValidConfig(config); err != nil {
		return nil, err
	}
	return &Command{
		Type:   "add",
		Source: config.Source,
		Sink:   config.Sink,
	}, nil
}

func CreateDatabase(dbName string, sink Sink) (*http.Response, error) {
	influxUrl := fmt.Sprintf("http://%s", sink)
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

func GetIpfsLogAddress(source Source) string {
	return fmt.Sprintf("http://%s/api/v0/log/tail?encoding=json&stream-channels=true", source)
}

func GetNodeId(source Source) (string, error) {
	url := fmt.Sprintf("http://%s/api/v0/id", source)
	resp, err := http.Get(url)
	if err != nil {
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
		return "", errors.New("Could not get nodeId, are you sure this is an ipfs daemon?")
	}
	return nodeId, nil
}
