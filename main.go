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
	Type     string              `json:"type"`     //add, remove, list
	Node     string              `json:"node"`     //the name of the node the command it for
	Source   []Source            `json:"source"`   //source of the log messages
	Sink     Sink                `json:"sink"`     //sink where the log messages will flow
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
			Usage: "Use Line Protocol Format (Influxdb) when writing to output instead of json",
		},
		cli.StringFlag{
			Name:  "config, c",
			Usage: "Specify a configuration file to use",
		},
	},
	Action: func(c *cli.Context) error {
		showUsage := func(w io.Writer) {
			fmt.Fprintln(w, "ipfs-metrics add -i [ip:port] -o [ip:port] [tagKey1=tagValue1...tagKeyn=tagValuen]\n")
			fmt.Fprintln(w, "ipfs-metrics add --config [configFile]\n")
		}
		cmd, err := NewAddCommand(c)
		if err != nil {
			showUsage(os.Stdout)
			return err
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
			Type: "remove",
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
			Type: "list",
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
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			handleConnection(w, r)
		})
		http.ListenAndServe(port, nil)
		return nil
	},
}
