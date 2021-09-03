package server

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/dictyBase/aphgrpc"
	manager "github.com/dictyBase/arangomanager"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	ontoarango "github.com/dictyBase/go-obograph/storage/arangodb"
	"github.com/dictyBase/modware-stock/internal/app/service"
	"github.com/dictyBase/modware-stock/internal/message/nats"
	"github.com/dictyBase/modware-stock/internal/repository/arangodb"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	gnats "github.com/nats-io/go-nats"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// RunServer starts and runs the server
func RunServer(c *cli.Context) error {
	srepo, err := arangodb.NewStockRepo(allParams(c))
	if err != nil {
		return cli.NewExitError(
			fmt.Sprintf("cannot connect to arangodb stocks repository %s", err.Error()),
			2,
		)
	}
	ms, err := nats.NewPublisher(
		c.String("nats-host"),
		c.String("nats-port"),
		gnats.MaxReconnects(-1),
		gnats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		return cli.NewExitError(
			fmt.Sprintf("cannot connect to messaging server %s", err.Error()),
			2,
		)
	}
	grpcS := grpc.NewServer(
		grpc_middleware.WithUnaryServerChain(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_logrus.UnaryServerInterceptor(getLogger(c)),
		),
	)
	stock.RegisterStockServiceServer(
		grpcS,
		service.NewStockService(
			srepo,
			ms,
			aphgrpc.TopicsOption(
				map[string]string{
					"stockCreate": "StockService.Create",
					"stockUpdate": "StockService.Update",
					"stockDelete": "StockService.Delete",
				}),
			strainType(c.String("strain-term")),
		),
	)
	if c.Bool("reflection") {
		// register reflection service on gRPC server
		reflection.Register(grpcS)
	}
	// create listener
	endP := fmt.Sprintf(":%s", c.String("port"))
	lis, err := net.Listen("tcp", endP)
	if err != nil {
		return cli.NewExitError(
			fmt.Sprintf("failed to listen %s", err),
			2,
		)
	}
	log.Printf("starting grpc server on %s", endP)
	grpcS.Serve(lis)
	return nil
}

func strainType(term string) aphgrpc.Option {
	return func(so *aphgrpc.ServiceOptions) {
		so.Params = map[string]string{"strain_term": term}
	}
}

func getLogger(c *cli.Context) *logrus.Entry {
	log := logrus.New()
	log.Out = os.Stderr
	switch c.GlobalString("log-format") {
	case "text":
		log.Formatter = &logrus.TextFormatter{
			TimestampFormat: "02/Jan/2006:15:04:05",
		}
	case "json":
		log.Formatter = &logrus.JSONFormatter{
			TimestampFormat: "02/Jan/2006:15:04:05",
		}
	}
	l := c.GlobalString("log-level")
	switch l {
	case "debug":
		log.Level = logrus.DebugLevel
	case "warn":
		log.Level = logrus.WarnLevel
	case "error":
		log.Level = logrus.ErrorLevel
	case "fatal":
		log.Level = logrus.FatalLevel
	case "panic":
		log.Level = logrus.PanicLevel
	}
	return logrus.NewEntry(log)
}

func allParams(c *cli.Context) (*manager.ConnectParams, *arangodb.CollectionParams, *ontoarango.CollectionParams) {
	arPort, _ := strconv.Atoi(c.String("arangodb-port"))
	connP := &manager.ConnectParams{
		User:     c.String("arangodb-user"),
		Pass:     c.String("arangodb-pass"),
		Database: c.String("arangodb-database"),
		Host:     c.String("arangodb-host"),
		Istls:    c.Bool("is-secure"),
		Port:     arPort,
	}
	collP := &arangodb.CollectionParams{
		Stock:              c.String("stock-collection"),
		StockProp:          c.String("stockprop-collection"),
		StockKeyGenerator:  c.String("stock-key-generator-collection"),
		StockType:          c.String("stock-type-edge"),
		ParentStrain:       c.String("parent-strain-edge"),
		StockPropTypeGraph: c.String("stockproptype-graph"),
		Strain2ParentGraph: c.String("strain2parent-graph"),
		StrainOntology:     c.String("strain-ontology"),
		KeyOffset:          c.Int("keyoffset"),
		StockTerm:          c.String("strain-term"),
		StockOntoGraph:     c.String("stockonto-graph"),
	}
	ontoP := &ontoarango.CollectionParams{
		GraphInfo:    c.String("cv-collection"),
		OboGraph:     c.String("obograph"),
		Relationship: c.String("rel-collection"),
		Term:         c.String("term-collection"),
	}
	return connP, collP, ontoP
}
