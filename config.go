package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	cli "github.com/codegangsta/cli"
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
	Source []Source `json:"Source"`
	Sink   Sink     `json:"Sink"`
}

func (s Sink) String() string {
	return fmt.Sprintf("%s:%s", s.Address, s.Port)
}
func (s Source) String() string {
	return fmt.Sprintf("%s:%s", s.Address, s.Port)
}
func (t Tag) String() string {
	return fmt.Sprintf("%s=%s", t.Name, t.Value)
}

//return nil if valid, error if else
func ValidConfig(config *Config) error {
	if !validSources(config.Source) {
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

func validSources(sources []Source) bool {
	for s := range sources {
		if len(sources[s].Address) == 0 || len(sources[s].Port) == 0 {
			return false
		}
	}
	return true
}

func MakeSink(format, address, port string) *Sink {
	return &Sink{
		Address: address,
		Port:    port,
		Format:  format,
	}
}

func MakeSource(address, port string, tags []Tag) *Source {
	return &Source{
		Address: address,
		Port:    port,
		Tags:    tags,
	}
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

func LoadConfigFromFile(path string) (*Config, error) {
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

func LoadConfigFromArgs(c *cli.Context) (*Config, error) {
	var config Config
	if len(c.String("input")) == 0 {
		return nil, errors.New("Input of event logs required")
	}
	input := strings.Split(c.String("input"), ":")
	if len(input) != 2 {
		return nil, errors.New("Input format invalid")
	}
	tags, err := MakeTags(c.Args())
	if err != nil {
		return nil, err
	}
	source := Source{
		Address: input[0],
		Port:    input[1],
		Tags:    tags,
	}
	config.Source = append(config.Source, source)

	var format string
	if c.Bool("lineprotocol") {
		format = "lineprotocol"
	} else {
		format = "json"
	}

	//Since sink is an optionl field
	var sink Sink
	if len(c.String("output")) == 0 {
		infolog.Println("No output given, will write to stdout")
		sink = Sink{
			Format: format,
		}
	} else {
		output := strings.Split(c.String("output"), ":")
		if len(input) != 2 {
			return nil, errors.New("Output format invalid")
		}
		sink = Sink{
			Address: output[0],
			Port:    output[1],
			Format:  format,
		}
	}
	config.Sink = sink
	return &config, nil
}
