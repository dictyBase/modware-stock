package server

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/dictyBase/apihelpers/aphgrpc"
	manager "github.com/dictyBase/arangomanager"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/app/service"
	"github.com/dictyBase/modware-stock/internal/message/nats"
	"github.com/dictyBase/modware-stock/internal/repository/arangodb"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	gnats "github.com/nats-io/go-nats"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	cli "gopkg.in/urfave/cli.v1"
)

// RunServer starts and runs the server
func RunServer(c *cli.Context) error {
	arPort, _ := strconv.Atoi(c.String("arangodb-port"))
	connP := &manager.ConnectParams{
		User:     c.String("arangodb-user"),
		Pass:     c.String("arangodb-pass"),
		Database: c.String("arangodb-database"),
		Host:     c.String("arangodb-host"),
		Port:     arPort,
		Istls:    c.Bool("is-secure"),
	}
	collP := &arangodb.CollectionParams{
		Stock:              c.String("stock-collection"),
		Strain:             c.String("strain-collection"),
		Plasmid:            c.String("plasmid-collection"),
		StockKeyGenerator:  c.String("stock-key-generator-collection"),
		StockPlasmid:       c.String("stock-plasmid-edge"),
		StockStrain:        c.String("stock-strain-edge"),
		ParentStrain:       c.String("parent-strain-edge"),
		Stock2PlasmidGraph: c.String("stock2plasmid-graph"),
		Stock2StrainGraph:  c.String("stock2strain-graph"),
		Strain2ParentGraph: c.String("strain2parent-graph"),
	}
	srepo, err := arangodb.NewStockRepo(connP, collP)
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
		),
	)
	reflection.Register(grpcS)

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
