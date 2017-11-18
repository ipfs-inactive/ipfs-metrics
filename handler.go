package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

//Handle request, add, remove, list
func handleConnection(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)
	cmd := &Command{}
	dec.Decode(cmd)
	cmd.Response = w

	var result = "Success"
	switch cmd.Cmd {
	case "add":
		err := handleAddCollection(cmd)
		if err != nil {
			result = err.Error()
		}
		cmd.Result = result
		printCmd, _ := json.MarshalIndent(cmd, "", "\t")
		fmt.Fprintf(cmd.Response, string(printCmd))
		return
	case "remove":
		err := handleRemoveCollection(cmd)
		if err != nil {
			result = err.Error()
		}
		cmd.Result = result
		printCmd, _ := json.MarshalIndent(cmd, "", "\t")
		fmt.Fprintf(cmd.Response, string(printCmd))
		return
	case "list":
		err := handleListCollection(cmd)
		if err != nil {
			result = err.Error()
		}
		break
	}
	return
}

//Remove a source from the collection
func handleRemoveCollection(cmd *Command) error {
	lp := proxyList[cmd.Source]
	if lp == nil {
		err := errors.New(fmt.Sprintf("ERROR - Source: %s not in collection", cmd.Source))
		return err //since this needs to go to the client
	}
	lp.Close()
	delete(proxyList, cmd.Source)
	return nil

}

type ListResult struct {
	Name   string   `json:"name"`
	Source string   `json:"source"`
	Sink   string   `json:"sink"`
	Format string   `json:"format"`
	Tags   []string `json:"tags"`
}

//List all sources in collection
func handleListCollection(cmd *Command) error {
	if proxyList != nil {
		for _, lp := range proxyList {
			lr := &ListResult{
				Name:   lp.Name,
				Source: lp.Source,
				Sink:   lp.Sink,
				Format: lp.Format,
				Tags:   lp.Tags,
			}
			ent, err := json.MarshalIndent(lr, "", "\t")
			if err != nil {
				panic(err)
			}
			fmt.Fprintf(cmd.Response, string(ent))
		}
	}
	return nil
}

//Add a source to the collection
func handleAddCollection(cmd *Command) error {
	if proxyList[cmd.Source] != nil {
		err := errors.New(fmt.Sprintf("ERROR - Source: %s already in collection", cmd.Source))
		return err
	}

	lp := &LogProxy{
		Name:     cmd.Source, //TODO make the tags a map and use the nodeId for the name
		Source:   cmd.Source,
		Sink:     cmd.Sink,
		Inbound:  make(chan LogEvent, 64),
		Outbound: make(chan LogEvent, 64),
		Tags:     cmd.Tags,
		Format:   cmd.Format,
	}

	go lp.Start()
	return nil
}
