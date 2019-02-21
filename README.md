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

## Subcommand

```
NAME:
   modware-stock start-server - starts the modware-stock microservice with grpc backends

USAGE:
   modware-stock start-server [command options] [arguments...]

OPTIONS:
   --port value                            tcp port at which the server will be available (default: "9560")
   --stock-collection value                arangodb collection for storing biological stocks (default: "stock")
   --strain-collection value               arangodb collection for storing strains (default: "strain")
   --plasmid-collection value              arangodb collection for storing plasmids (default: "plasmid")
   --stock-key-generator-collection value  arangodb collection for generating unique IDs (default: "stock_key_generator")
   --stock-plasmid-edge value              arangodb edge collection for connecting stocks and plasmids (default: "stock_plasmid")
   --stock-strain-edge value               arangodb edge collection for connecting stocks and strains (default: "stock_strain")
   --parent-strain-edge value              arangodb edge collection for connecting strains to their parents (default: "parent_strain")
   --stock2plasmid-graph value             arangodb named graph for managing relations between stocks and plasmids (default: "stock2plasmid")
   --stock2strain-graph value              arangodb named graph for managing relations between stocks and strains (default: "stock2strain")
   --strain2parent-graph value             arangodb named graph for managing relations between strains and their parents (default: "strain2parent")
   --arangodb-pass value, --pass value     arangodb database password [$ARANGODB_PASS]
   --arangodb-database value, --db value   arangodb database name [$ARANGODB_DATABASE]
   --arangodb-user value, --user value     arangodb database user [$ARANGODB_USER]
   --arangodb-host value, --host value     arangodb database host (default: "arangodb") [$ARANGODB_SERVICE_HOST]
   --arangodb-port value                   arangodb database port (default: "8529") [$ARANGODB_SERVICE_PORT]
   --is-secure                             flag for secured or unsecured arangodb endpoint
   --nats-host value                       nats messaging server host [$NATS_SERVICE_HOST]
   --nats-port value                       nats messaging server port [$NATS_SERVICE_PORT]
   --reflection, --ref                     flag for enabling server reflection
```

# API

### gRPC

The protocol buffer definitions and service apis are documented
[here](https://github.com/dictyBase/dictybaseapis/blob/master/dictybase/stock/stock.proto).
