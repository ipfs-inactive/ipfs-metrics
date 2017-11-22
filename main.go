package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	cli "github.com/codegangsta/cli"
)

var infolog, errlog *log.Logger
var port, db string
var proxyList = make(map[string]*LogProxy)

type Command struct {
	Cmd      string              `json:"cmds"`     //add, remove, list
	Node     string              `json:"node"`     //the name of the node the command it for
	Source   string              `json:"source"`   //source of the log messages
	Sink     string              `json:"sink"`     //sink where the log messages will flow
	Tags     []string            `json:"tags"`     //tags on log messages, nodeId is added by default (serves as a liveness check)
	Format   string              `json:"format"`   //json or line protocol
	Result   string              `json:"result"`   //result of command - success or error message
	Response http.ResponseWriter `json:"response"` //where the result of the command will be written
}

func init() {
	infolog = log.New(os.Stderr, "INFO - ", log.Ldate|log.Ltime)
	errlog = log.New(os.Stderr, "ERROR - ", log.Ldate|log.Ltime)
	port = ":9123"
	db = "ipfsmetrics"
}

func main() {
	app := cli.NewApp()
	app.Usage = "ipfs-metrics is a tool for working with ipfs events"
	app.Commands = []cli.Command{
		startCmd,
		addCmd,
		rmCmd,
		listCmd,
	}
	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var addCmd = cli.Command{
	Name:  "add",
	Usage: "add an ipfs daemon to metrics collection",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "input, i",
			Usage: "Input of the event logs",
		},
		cli.StringFlag{
			Name:  "output, o",
			Usage: "Output to which the event logs will flow (if empty will use stdout)",
		},
		cli.BoolFlag{
			Name:  "lineprotocol, lp",
			Usage: "Use Line Protocol Format (Influxdb) when writing to output",
		},
	},
	Action: func(c *cli.Context) error {
		showUsage := func(w io.Writer) {
			fmt.Fprintln(w, "ipfs-metrics add -i [input - ip:port] -o [output ip:port] [tagKey1=tagValue1...tagKeyn=tagValuen]\n")
		}
		var source, sink string
		var tags []string
		var format string

		if len(c.String("input")) == 0 {
			fmt.Printf("Source of event logs required: '%s add -i 127.0.0.1:5001'\n", os.Args[0])
			showUsage(os.Stdout)
			os.Exit(1)
		} else {
			source = c.String("input")
		}

		//TODO: Use enums for stdout and protocol
		if len(c.String("output")) == 0 {
			infolog.Println("No output given, will write to stdout")
			sink = "stdout"
		} else {
			sink = c.String("output")
		}

		if c.Bool("lineprotocol") {
			format = "lineProtocol"
		} else {
			format = "json"
		}

		//A default Tag, also a liveness check for ipfs daemon
		nodeId, err := GetNodeId(source)
		if err != nil {
			return err
		}
		tags = append(tags, fmt.Sprintf("nodeId=%s", nodeId))
		//Tail does not return the first argument after a flag, bug maybe, or imporper usage?
		//TODO some type of tag validation..something something - doesn't contain equal sign..
		//		tags = append(tags, c.Args().Get(0))
		tags = append(tags, c.Args().Tail()...)
		cmd := &Command{
			Cmd:    "add",
			Node:   nodeId,
			Source: source,
			Sink:   sink,
			Tags:   tags,
			Format: format,
		}
		resp, err := SendCommand(cmd)
		if err != nil {
			errlog.Fatal("Please run `ipfs-metrics start` first")
			os.Exit(1)
		}
		io.Copy(os.Stdout, resp.Body)
		return nil
	},
}

var rmCmd = cli.Command{
	Name:  "remove",
	Usage: "remove ipfs daemon from metrics collection",
	Action: func(c *cli.Context) error {
		cmd := &Command{
			Cmd:  "remove",
			Node: c.Args().First(),
		}
		resp, err := SendCommand(cmd)
		if err != nil {
			errlog.Fatal("Please run `ipfs-metrics start` first")
			os.Exit(1)
		}
		io.Copy(os.Stdout, resp.Body)
		return nil
	},
}

var listCmd = cli.Command{
	Name:  "list",
	Usage: "list ipfs daemons in metrics collection",
	Action: func(c *cli.Context) error {
		cmd := &Command{
			Cmd: "list",
		}
		resp, err := SendCommand(cmd)
		if err != nil {
			errlog.Fatal("Please run `ipfs-metrics start` first")
			os.Exit(1)
		}
		io.Copy(os.Stdout, resp.Body)
		return nil
	},
}

var startCmd = cli.Command{
	Name:  "start",
	Usage: "starts ipfs-metricsd",
	Action: func(c *cli.Context) error {
		infolog.Println("ipfs-metricsd starting...")
		infolog.Print("Ensuring database exists...")
		_, err := CreateDatabase(db)
		if err != nil {
			errlog.Println("Failed to create database: ", err)
			errlog.Fatal("Please ensure that influxdb is running")
		}
		infolog.Print("database found!")
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			handleConnection(w, r)
		})
		http.ListenAndServe(port, nil)
		return nil
	},
}
