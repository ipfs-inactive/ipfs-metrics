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
)

func NewCommand(cmd, source, sink string, tags []string) *Command {
	return &Command{
		Cmd:    cmd,
		Source: source,
		Sink:   sink,
		Tags:   tags,
	}
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
