package main

import (
	"os"

	apiflag "github.com/dictyBase/apihelpers/command/flag"
	arangoflag "github.com/dictyBase/arangomanager/command/flag"
	"github.com/dictyBase/modware-stock/internal/app/server"
	"github.com/dictyBase/modware-stock/internal/app/validate"
	cli "gopkg.in/urfave/cli.v1"
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
			Flags:  getServerFlags(),
		},
	}
	app.Run(os.Args)
}

func getServerFlags() []cli.Flag {
	var f []cli.Flag
	f = append(
		f,
		[]cli.Flag{
			cli.StringFlag{
				Name:  "port",
				Usage: "tcp port at which the server will be available",
				Value: "9560",
			},
			cli.StringFlag{
				Name:  "stock-collection",
				Usage: "arangodb collection for storing biological stocks",
				Value: "stock",
			},
			cli.StringFlag{
				Name:  "strain-collection",
				Usage: "arangodb collection for storing strains",
				Value: "strain",
			},
			cli.StringFlag{
				Name:  "plasmid-collection",
				Usage: "arangodb collection for storing plasmids",
				Value: "plasmid",
			},
			cli.StringFlag{
				Name:  "stock-key-generator-collection",
				Usage: "arangodb collection for generating unique IDs",
				Value: "stock_key_generator",
			},
			cli.StringFlag{
				Name:  "stock-plasmid-edge",
				Usage: "arangodb edge collection for connecting stocks and plasmids",
				Value: "stock_plasmid",
			},
			cli.StringFlag{
				Name:  "stock-strain-edge",
				Usage: "arangodb edge collection for connecting stocks and strains",
				Value: "stock_strain",
			},
			cli.StringFlag{
				Name:  "parent-strain-edge",
				Usage: "arangodb edge collection for connecting strains to their parents",
				Value: "parent_strain",
			},
			cli.StringFlag{
				Name:  "stock2plasmid-graph",
				Usage: "arangodb named graph for managing relations between stocks and plasmids",
				Value: "stock2plasmid",
			},
			cli.StringFlag{
				Name:  "stock2strain-graph",
				Usage: "arangodb named graph for managing relations between stocks and strains",
				Value: "stock2strain",
			},
			cli.StringFlag{
				Name:  "strain2parent-graph",
				Usage: "arangodb named graph for managing relations between strains and their parents",
				Value: "strain2parent",
			},
			cli.BoolTFlag{
				Name:  "reflection, ref",
				Usage: "flag for enabling server reflection",
			},
		}...,
	)
	f = append(f, arangoflag.ArangodbFlags()...)
	f = append(f, apiflag.NatsFlag()...)
	return f
}
