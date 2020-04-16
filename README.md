## This repository has been archived!

*This IPFS-related repository has been archived, and all issues are therefore frozen*. If you want to ask a question or open/continue a discussion related to this repo, please visit the [official IPFS forums](https://discuss.ipfs.io).

We archive repos for one or more of the following reasons:

- Code or content is unmaintained, and therefore might be broken
- Content is outdated, and therefore may mislead readers
- Code or content evolved into something else and/or has lived on in a different place
- The repository or project is not active in general

Please note that in order to keep the primary IPFS GitHub org tidy, most archived repos are moved into the [ipfs-inactive](https://github.com/ipfs-inactive) org.

If you feel this repo should **not** be archived (or portions of it should be moved to a non-archived repo), please [reach out](https://ipfs.io/help) and let us know. Archiving can always be reversed if needed.

---
   
# ipfs-metrics (pre-alpha)
`ipfs-metrics` is a program used to view, filter and manage event logs produced by IPFS nodes. Connect 1 or more IPFS nodes to `ipfs-metrics`, add tags to their event logs, choose the format (json or [line protocol](https://docs.influxdata.com/influxdb/v1.3/write_protocols/line_protocol_tutorial/)) the logs are displayed in, view the events in stdout or direct them to an http endpoint. `ipfs-metrics` hopes to make managing IPFS event logs easy. 

### Install
```
go get github.com/ipfs/ipfs-metrics
```

### Setup
```
$ cd ipfs-metrics/docker
$ docker-compose up -d
$ ipfs daemon
```

### Usage
```
$ ipfs-metric --help

NAME:
   ipfs-metrics - ipfs-metrics is a tool for working with ipfs events

USAGE:
   ipfs-metrics [global options] command [command options] [arguments...]

VERSION:
   0.0.0

COMMANDS:
     start    starts ipfs-metricsd
     add      add an ipfs daemon to metrics collection
     remove   remove ipfs daemon from metrics collection
     list     list ipfs daemons in metrics collection
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h     show help
   --version, -v  print the version
```

### Example
```
$ ipfs-metrics start
INFO - 2017/11/17 15:00:20 ipfs-metricsd starting...

$ ipfs-metrics add --lineprotocol -i "127.0.0.1:5001" -o "127.0.0.1:8086"
INFO - 2017/11/17 15:01:18 ipfs-metricsd starting...
INFO - 2017/11/17 15:01:19 Writer Open Out-Stream: 127.0.0.2:8086
INFO - 2017/11/17 15:01:19 Reader Open In-Stream: 127.0.0.1:5001
INFO - 2017/11/17 15:01:19 Filter Open In-Stream: 127.0.0.1:5001

$ ipfs-metrics add --lineprotocol -i "127.0.0.2:5001" -o "127.0.0.1:8086" "Tag=ImATag"
INFO - 2017/11/17 15:01:18 ipfs-metricsd starting...
INFO - 2017/11/17 15:01:19 Writer Open Out-Stream: 127.0.0.1:8086
INFO - 2017/11/17 15:01:19 Reader Open In-Stream: 127.0.0.2:5001
INFO - 2017/11/17 15:01:19 Filter Open In-Stream: 127.0.0.2:5001

$ ipfs-metrics list
{
        "name": "127.0.0.1:5001",
        "source": "127.0.0.1:5001",
        "sink": "127.0.0.1:8086",
        "format": "lineProtocol",
        "tags": [
                "nodeId=QmcJ9RHiEoa1WYeaFAEHVgjc41aXfD52WDEFLZrEcQvbPR"
        ]
}
{
        "name": "127.0.0.2:5001",
        "source": "127.0.0.1:5001",
        "sink": "127.0.0.1:8086",
        "format": "lineProtocol",
        "tags": [
                "nodeId=QmcJ9RHiEoa1WYeaFAEHVgjc41aXfD52WDEFLZrEcQvbPR",
                "Tag=ImATag", //added each log event
        ]
}

$ ipfs-metrics remove 127.0.0.1:5001
Closing Connection: 127.0.0.1:5001
INFO - 2017/11/17 15:04:59 Writer Close Out-Stream: 127.0.0.1:8086
INFO - 2017/11/17 15:04:59 Filter Close In-Stream: 127.0.0.1:5001
INFO - 2017/11/17 15:04:59 Reader Close In-Stream: 127.0.0.1:5001

$ ipfs-metrics remove 127.0.0.2:5001
INFO - 2017/11/17 15:04:59 Writer Close Out-Stream: 127.0.0.2:8086
INFO - 2017/11/17 15:04:59 Filter Close In-Stream: 127.0.0.2:5001
INFO - 2017/11/17 15:04:59 Reader Close In-Stream: 127.0.0.2:5001
```

### License
MIT
