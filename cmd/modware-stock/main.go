package main

import (
	"fmt"
	"os"

	"github.com/dictyBase/aphgrpc"
	arango "github.com/dictyBase/arangomanager/command/flag"
	oboflag "github.com/dictyBase/go-obograph/command/flag"
	"github.com/dictyBase/modware-stock/internal/app/server"
	"github.com/dictyBase/modware-stock/internal/app/validate"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "modware-stock"
	app.Usage = "cli for modware-stock microservice"
	app.Version = "1.0.0"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "log-format",
			Usage: "format of the logging out, either of json or text.",
			Value: "json",
		},
		cli.StringFlag{
			Name:  "log-level",
			Usage: "log level for the application",
			Value: "error",
		},
	}
	app.Commands = []cli.Command{
		{
			Name:   "start-server",
			Usage:  "starts the modware-stock microservice with grpc backends",
			Action: server.RunServer,
			Before: validate.ValidateServerArgs,
			Flags:  allFlags(),
		},
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Printf("error in running the app %s", err)
		os.Exit(1)
	}
}

func allFlags() []cli.Flag {
	f := make([]cli.Flag, 0)
	f = append(f, serverFlags()...)
	f = append(f, dbCollectionFlags()...)
	f = append(f, arango.ArangoFlags()...)
	f = append(f, []cli.Flag{
		cli.StringFlag{
			Name:   "arangodb-database, db",
			EnvVar: "ARANGODB_DATABASE",
			Usage:  "arangodb database name",
			Value:  "stock",
		},
	}...)
	f = append(f, oboflag.OntologyFlagsOnly()...)
	return append(f, aphgrpc.NatsFlag()...)
}

func serverFlags() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:  "port",
			Usage: "tcp port at which the server will be available",
			Value: "9560",
		},
		cli.IntFlag{
			Name:  "keyoffset",
			Usage: "initial offset for stock id generation",
			Value: 370000,
		},
		cli.BoolTFlag{
			Name:  "reflection, ref",
			Usage: "flag for enabling server reflection",
		},
		cli.StringFlag{
			Name:  "strain-ontology",
			Usage: "dictybase ontology that will be used for picking grouping term for strain",
			Value: "dicty_strain_property",
		},
		cli.StringFlag{
			Name:  "strain-term",
			Usage: "default ontology term that will be used for creating strain",
			Value: "general strain",
		},
	}
}

func dbCollectionFlags() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:  "stock-collection",
			Usage: "arangodb collection for storing biological stocks",
			Value: "stock",
		},
		cli.StringFlag{
			Name:  "stockprop-collection",
			Usage: "arangodb collection for storing stock properties",
			Value: "stockprop",
		},
		cli.StringFlag{
			Name:  "stock-key-generator-collection",
			Usage: "arangodb collection for generating unique IDs",
			Value: "stock_key_generator",
		},
		cli.StringFlag{
			Name:  "stock-type-edge",
			Usage: "arangodb edge collection for connecting stocks to their types (strain or plasmid)",
			Value: "stock_type",
		},
		cli.StringFlag{
			Name:  "parent-strain-edge",
			Usage: "arangodb edge collection for connecting strains to their parent",
			Value: "parent_strain",
		},
		cli.StringFlag{
			Name:  "stock-term-edge",
			Usage: "arangodb edge collection for connecting stock to ontology term",
			Value: "stock_term",
		},
		cli.StringFlag{
			Name:  "stockproptype-graph",
			Usage: "arangodb named graph for managing relations between stocks and their properties",
			Value: "stockprop_type",
		},
		cli.StringFlag{
			Name:  "strain2parent-graph",
			Usage: "arangodb named graph for managing relations between strains and their parents",
			Value: "strain2parent",
		},
		cli.StringFlag{
			Name:  "stockonto-graph",
			Usage: "arangodb named graph for managing stock and ontology",
			Value: "stockonto",
		},
	}
}
