package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
)

type Tag struct {
	Name  string `json:"Name"`
	Value string `json:"Value"`
}

type Source struct {
	Address string `json:"Address"`
	Port    string `json:"Port"`
	Tags    []Tag  `json:"Tags"`
}
type Sink struct {
	Address string `json:"Address"`
	Port    string `json:"Port"`
	Format  string `json:"Format"`
}

type Config struct {
	Source Source `json:"Source"`
	Sink   Sink   `json:"Sink"`
}

func MakeTag(name, value string) Tag {
	return Tag{
		Name:  name,
		Value: value,
	}
}

func ValidTag(maybeTag string) bool {
	if !strings.Contains(maybeTag, "=") {
		return false
	}
	if strings.Contains(maybeTag, ",") {
		return false
	}
	if strings.HasSuffix(maybeTag, `\`) {
		return false
	}
	nv := strings.Split(maybeTag, "=")
	if len(nv) != 2 {
		return false
	}
	if len(nv[0]) == 0 || len(nv[1]) == 0 {
		return false
	}
	return true
}

func MakeTags(tags []string) ([]Tag, error) {
	var ts []Tag
	for t := range tags {
		if !ValidTag(tags[t]) {
			return nil, errors.New(fmt.Sprintf("Invalid tag: %v", tags[t]))
		}
		tmp := strings.Split(tags[t], "=")
		//validateTag garentees this won't panic
		ts = append(ts, MakeTag(tmp[0], tmp[1]))
	}
	return ts, nil
}

func LoadConfig(path string) (*Config, error) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	err = json.Unmarshal(file, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}
