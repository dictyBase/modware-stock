// Package server provides the gRPC and HTTP gateway server setup and lifecycle management for the stock service.
package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	E "github.com/IBM/fp-go/v2/either"
	F "github.com/IBM/fp-go/v2/function"
	IO "github.com/IBM/fp-go/v2/io"
	IOE "github.com/IBM/fp-go/v2/ioeither"
	L "github.com/IBM/fp-go/v2/optics/lens"
	O "github.com/IBM/fp-go/v2/option"
	P "github.com/IBM/fp-go/v2/predicate"
	R "github.com/IBM/fp-go/v2/record"
	"github.com/dictyBase/aphgrpc"
	manager "github.com/dictyBase/arangomanager"
	eitherparse "github.com/dictyBase/fp-go-loom/either/parse"
	ioeutils "github.com/dictyBase/fp-go-loom/ioeitherutils"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	ontoarango "github.com/dictyBase/go-obograph/storage/arangodb"
	"github.com/dictyBase/modware-stock/internal/app/service"
	"github.com/dictyBase/modware-stock/internal/message"
	"github.com/dictyBase/modware-stock/internal/message/nats"
	"github.com/dictyBase/modware-stock/internal/repository"
	"github.com/dictyBase/modware-stock/internal/repository/arangodb"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	gnats "github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type serverState struct {
	cliCtx   *cli.Context
	repo     repository.StockRepository
	pub      message.Publisher
	grpcS    *grpc.Server
	stockSvc *service.StockService
	httpS    *http.Server
	lis      net.Listener
}

const logTimestampFormat = "02/Jan/2006:15:04:05"

var (
	serverRepoLens = L.MakeLens(
		func(s serverState) repository.StockRepository { return s.repo },
		func(s serverState, repo repository.StockRepository) serverState {
			s.repo = repo
			return s
		},
	)
	serverPubLens = L.MakeLens(
		func(s serverState) message.Publisher { return s.pub },
		func(s serverState, pub message.Publisher) serverState {
			s.pub = pub
			return s
		},
	)
	serverGRPCServerLens = L.MakeLens(
		func(s serverState) *grpc.Server { return s.grpcS },
		func(s serverState, grpcS *grpc.Server) serverState {
			s.grpcS = grpcS
			return s
		},
	)
	serverStockSvcLens = L.MakeLens(
		func(s serverState) *service.StockService { return s.stockSvc },
		func(s serverState, stockSvc *service.StockService) serverState {
			s.stockSvc = stockSvc
			return s
		},
	)
	serverHTTPServerLens = L.MakeLens(
		func(s serverState) *http.Server { return s.httpS },
		func(s serverState, httpS *http.Server) serverState {
			s.httpS = httpS
			return s
		},
	)
	serverListenerLens = L.MakeLens(
		func(s serverState) net.Listener { return s.lis },
		func(s serverState, lis net.Listener) serverState {
			s.lis = lis
			return s
		},
	)

	maybeRegisterReflection = F.Pipe1(
		hasReflectionFlag,
		P.Fold(F.Identity[serverState], registerReflectionOnState),
	)

	logFormatters = map[string]func() logrus.Formatter{
		logFormatText: newTextFormatter,
		logFormatJSON: newJSONFormatter,
	}
	logLevels = map[string]logrus.Level{
		"debug":       logrus.DebugLevel,
		"warn":        logrus.WarnLevel,
		logLevelError: logrus.ErrorLevel,
		"fatal":       logrus.FatalLevel,
		"panic":       logrus.PanicLevel,
	}
)

func openStockRepository(
	s serverState,
) IOE.IOEither[error, repository.StockRepository] {
	connP, collP, ontoP := allParams(s.cliCtx)
	return F.Pipe1(
		IOE.TryCatchError(func() (repository.StockRepository, error) {
			return arangodb.NewStockRepo(connP, collP, ontoP)
		}),
		IOE.MapLeft[repository.StockRepository](
			exitErrorWithPrefix("cannot connect to arangodb stocks repository"),
		),
	)
}

func openStockPublisher(
	s serverState,
) IOE.IOEither[error, message.Publisher] {
	return F.Pipe1(
		IOE.TryCatchError(func() (message.Publisher, error) {
			return nats.NewPublisher(
				s.cliCtx.String("nats-host"),
				s.cliCtx.String("nats-port"),
				gnats.MaxReconnects(-1),
				gnats.ReconnectWait(2*time.Second),
			)
		}),
		IOE.MapLeft[message.Publisher](
			exitErrorWithPrefix("cannot connect to messaging server"),
		),
	)
}

func buildGRPCServer(s serverState) *grpc.Server {
	logger := getLogger(s.cliCtx)
	return grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_logrus.UnaryServerInterceptor(logger),
		),
	)
}

func buildStockService(s serverState) *service.StockService {
	return service.NewStockService(
		s.repo,
		s.pub,
		aphgrpc.TopicsOption(
			map[string]string{
				"stockCreate": "StockService.Create",
				"stockUpdate": "StockService.Update",
				"stockDelete": "StockService.Delete",
			}),
		strainType(s.cliCtx.String("strain-term")),
		plasmidType(s.cliCtx.String("plasmid-term")),
	)
}

// hasReflectionFlag reports whether the server CLI context asks for
// gRPC reflection registration.
//
// fp-go:allow-trivial-delegate cli.Context exposes the flag only
// through its Bool method, so the predicate cannot compose from a
// field extractor.
func hasReflectionFlag(s serverState) bool {
	return s.cliCtx.Bool("reflection")
}

func registerReflectionOnState(s serverState) serverState {
	reflection.Register(s.grpcS)
	return s
}

func registerGRPCServices(s serverState) IO.IO[struct{}] {
	return func() struct{} {
		stock.RegisterStockServiceServer(s.grpcS, s.stockSvc)
		_ = maybeRegisterReflection(s)
		return struct{}{}
	}
}

func buildGatewayServerFromState(
	s serverState,
) IOE.IOEither[error, *http.Server] {
	in := gatewayInput{
		ctx:      context.Background(),
		grpcS:    s.grpcS,
		stockSvc: s.stockSvc,
	}
	return newMultiplexedServer(in)
}

func listenServerFromState(
	s serverState,
) IOE.IOEither[error, net.Listener] {
	endP := fmt.Sprintf(":%s", s.cliCtx.String("port"))
	return F.Pipe1(
		IOE.TryCatchError(func() (net.Listener, error) {
			return net.Listen("tcp", endP)
		}),
		IOE.MapLeft[net.Listener](exitErrorWithPrefix("failed to listen")),
	)
}

func serveHTTPFromState(
	s serverState,
) IOE.IOEither[error, serverState] {
	endP := fmt.Sprintf(":%s", s.cliCtx.String("port"))
	return F.Pipe2(
		IOE.TryCatchError(func() (struct{}, error) {
			log.Printf("starting grpc and http gateway server on %s", endP)
			return struct{}{}, s.httpS.Serve(s.lis)
		}),
		IOE.MapLeft[struct{}](func(err error) error {
			return fmt.Errorf("error in running server %s", err)
		}),
		IOE.Map[error](F.Constant1[struct{}](s)),
	)
}

// RunServer starts and runs the multiplexed gRPC and HTTP gateway server.
func RunServer(c *cli.Context) error {
	return F.Pipe11(
		serverState{cliCtx: c},
		IOE.Of[error, serverState],
		IOE.Bind(serverRepoLens.Set, openStockRepository),
		IOE.Bind(serverPubLens.Set, openStockPublisher),
		IOE.Let[error](serverGRPCServerLens.Set, buildGRPCServer),
		IOE.Let[error](serverStockSvcLens.Set, buildStockService),
		IOE.ChainFirstIOK[error](registerGRPCServices),
		IOE.Bind(serverHTTPServerLens.Set, buildGatewayServerFromState),
		IOE.Bind(serverListenerLens.Set, listenServerFromState),
		IOE.Chain(serveHTTPFromState),
		ioeutils.ToEither[error, serverState],
		E.ToError[serverState],
	)
}

func setServiceParam(key, term string) aphgrpc.Option {
	return func(so *aphgrpc.ServiceOptions) {
		so.Params = F.Pipe1(so.Params, R.UpsertAt(key, term))
	}
}

func strainType(term string) aphgrpc.Option {
	return setServiceParam("strain_term", term)
}

func plasmidType(term string) aphgrpc.Option {
	return setServiceParam("plasmid_term", term)
}

func newTextFormatter() logrus.Formatter {
	return &logrus.TextFormatter{
		TimestampFormat: logTimestampFormat,
	}
}

func newJSONFormatter() logrus.Formatter {
	return &logrus.JSONFormatter{
		TimestampFormat: logTimestampFormat,
	}
}

func applyLoggerFormatter(logger *logrus.Logger, format string) {
	setFormatter := func(factory func() logrus.Formatter) logrus.Formatter {
		logger.Formatter = factory()
		return logger.Formatter
	}
	_ = F.Pipe2(
		logFormatters,
		R.Lookup[func() logrus.Formatter](format),
		O.Map(setFormatter),
	)
}

func applyLoggerLevel(logger *logrus.Logger, level string) {
	setLevel := func(lvl logrus.Level) logrus.Level {
		logger.Level = lvl
		return lvl
	}
	_ = F.Pipe2(
		logLevels,
		R.Lookup[logrus.Level](level),
		O.Map(setLevel),
	)
}

func getLogger(c *cli.Context) *logrus.Entry {
	logger := logrus.New()
	logger.Out = os.Stderr
	applyLoggerFormatter(logger, c.GlobalString(flagLogFormat))
	applyLoggerLevel(logger, c.GlobalString(flagLogLevel))
	return logrus.NewEntry(logger)
}

func parseArangoPort(portStr string) int {
	return F.Pipe2(
		portStr,
		eitherparse.ParseInt,
		E.GetOrElse(F.Constant1[error](0)),
	)
}

func allParams(
	c *cli.Context,
) (*manager.ConnectParams, *arangodb.CollectionParams, *ontoarango.CollectionParams) {
	arPort := parseArangoPort(c.String(flagArangodbPort))
	connP := &manager.ConnectParams{
		User:     c.String("arangodb-user"),
		Pass:     c.String("arangodb-pass"),
		Database: c.String("arangodb-database"),
		Host:     c.String("arangodb-host"),
		Istls:    c.Bool(flagIsSecure),
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
		PlasmidOntology:    c.String("plasmid-ontology"),
		KeyOffset:          c.Int("keyoffset"),
		StockTerm:          c.String("stock-term-edge"),
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
