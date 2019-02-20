# modware-stock

[![GoDoc](https://godoc.org/github.com/dictybase/modware-stock?status.svg)](https://godoc.org/github.com/dictybase/modware-stock)
[![Go Report Card](https://goreportcard.com/badge/github.com/dictybase/modware-stock)](https://goreportcard.com/report/github.com/dictybase/modware-stock)

dictyBase API server to manage biological stocks. The API server supports gRPC protocol for data exchange.

## Usage

```
NAME:
   modware-stock - cli for modware-stock microservice

USAGE:
   modware-stock [global options] command [command options] [arguments...]

VERSION:
   1.0.0

COMMANDS:
     start-server  starts the modware-stock microservice with grpc backends
     help, h       Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --log-format value  format of the logging out, either of json or text. (default: "json")
   --log-level value   log level for the application (default: "error")
   --help, -h          show help
   --version, -v       print the version
```

# API

### gRPC

The protocol buffer definitions and service apis are documented
[here](https://github.com/dictyBase/dictybaseapis/blob/master/dictybase/stock/stock.proto).
