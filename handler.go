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
	switch cmd.Type {
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
		err := handleRemoveCollection(cmd.Node)
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
func handleRemoveCollection(node string) error {
	lp := proxyList[node]
	if lp == nil {
		err := errors.New(fmt.Sprintf("ERROR - Source: %s not in collection", node))
		return err //since this needs to go to the client
	}
	lp.Close()
	delete(proxyList, node)
	return nil

}

type ListResult struct {
	Name   string `json:"name"`
	Source Source `json:"source"`
	Sink   Sink   `json:"sink"`
	Format string `json:"format"`
	Tags   []Tag  `json:"tags"`
}

//List all sources in collection
func handleListCollection(cmd *Command) error {
	if proxyList != nil {
		for _, lp := range proxyList {
			lr := &ListResult{
				Name:   lp.Name,
				Source: lp.Source,
				Sink:   lp.Sink,
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
	//start a routine for each source, if there is an error with one, skip it
	for s := range cmd.Source {
		source := cmd.Source[s]
		name, err := GetNodeId(source)
		if err != nil {
			//TODO Add feature to start and stop collection on different sources
			//e.g. add the source with status offline and poll it till its up/producing logs
			fmt.Fprintf(cmd.Response, "Failed to get NodeId: "+err.Error()+" will skip")
			continue
		}
		//we do not want to add the same source twice
		if proxyList[name] != nil {
			err := fmt.Sprintf("Source: %s, with Name: %s already in collection, will skip", source, name)
			fmt.Fprintf(cmd.Response, err)
			continue
		}
		source.Tags = append(source.Tags, MakeTag("nodeId", name))
		lp := &LogProxy{
			Name:     name,
			Source:   source,
			Sink:     cmd.Sink,
			Inbound:  make(chan LogEvent, 64),
			Outbound: make(chan LogEvent, 64),
		}
		go lp.Start()
	}

	return nil
}
