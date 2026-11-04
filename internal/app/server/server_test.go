package server

import (
	"context"
	"flag"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	E "github.com/IBM/fp-go/v2/either"
	F "github.com/IBM/fp-go/v2/function"
	"github.com/alicebob/miniredis/v2"
	"github.com/dictyBase/aphgrpc"
	ioeutils "github.com/dictyBase/fp-go-loom/ioeitherutils"
	loomcheck "github.com/dictyBase/fp-go-loom/loomcheck"
	"github.com/dictyBase/fp-go-loom/pipelinecheck"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/app/middleware"
	"github.com/dictyBase/modware-stock/internal/cache"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	empty "google.golang.org/protobuf/types/known/emptypb"
)

type stubGatewayStockServer struct {
	stock.UnimplementedStockServiceServer
	countCalls *atomic.Int32
}

// runServerIntegrationTestDoc provides comprehensive documentation for testing RunServer.
// This documentation describes the integration testing requirements for the RunServer function.
//
// The RunServer function performs the following operations:
// 1. Connects to ArangoDB using arangodb.NewStockRepo()
// 2. Connects to NATS messaging server using nats.NewPublisher()
// 3. Creates a gRPC server with logging interceptors
// 4. Registers the StockService with the gRPC server
// 5. Optionally enables gRPC reflection
// 6. Starts listening on a TCP port
// 7. Serves gRPC requests
//
// To properly test this function, you would need:
//
// 1. ArangoDB Test Instance:
//   - Use testcontainers-go to spin up an ArangoDB container
//   - Initialize required collections and graphs
//   - Provide connection parameters via cli.Context
//
// 2. NATS Test Instance:
//   - Use testcontainers-go/modules/nats to spin up a NATS container
//   - Provide NATS host and port via cli.Context
//
// 3. Test Strategy:
//   - Happy path: Server starts successfully with valid dependencies
//   - Error path: ArangoDB connection failure
//   - Error path: NATS connection failure
//   - Error path: Port already in use
//   - Test gRPC reflection enabled/disabled
//
// 4. Dependencies needed:
//   - github.com/testcontainers/testcontainers-go
//   - github.com/testcontainers/testcontainers-go/modules/nats
//   - ArangoDB testcontainer (custom or community module)
//
// 5. Reference existing integration tests:
//   - See internal/message/nats/nats_integration_test.go for NATS integration testing
//   - See internal/repository/arangodb/*_test.go for ArangoDB setup patterns

func TestStrainType(t *testing.T) {
	t.Parallel()

	t.Run("SetsStrainTermInEmptyParams", func(t *testing.T) {
		testStrainTypeWithEmptyParams(t)
	})

	t.Run("SetsStrainTermInExistingParams", func(t *testing.T) {
		testStrainTypeWithExistingParams(t)
	})

	t.Run("OverwritesExistingStrainTerm", func(t *testing.T) {
		testStrainTypeOverwritesExisting(t)
	})

	t.Run("WorksWithEmptyString", func(t *testing.T) {
		testStrainTypeWithEmptyString(t)
	})
}

func TestPlasmidType(t *testing.T) {
	t.Parallel()

	t.Run("SetsPlasmidTermInEmptyParams", func(t *testing.T) {
		testPlasmidTypeWithEmptyParams(t)
	})

	t.Run("SetsPlasmidTermInExistingParams", func(t *testing.T) {
		testPlasmidTypeWithExistingParams(t)
	})

	t.Run("OverwritesExistingPlasmidTerm", func(t *testing.T) {
		testPlasmidTypeOverwritesExisting(t)
	})

	t.Run("WorksWithEmptyString", func(t *testing.T) {
		testPlasmidTypeWithEmptyString(t)
	})
}

func TestGetLogger(t *testing.T) {
	t.Parallel()

	t.Run("CreatesLoggerWithTextFormat", func(t *testing.T) {
		testGetLoggerWithTextFormat(t)
	})

	t.Run("CreatesLoggerWithJSONFormat", func(t *testing.T) {
		testGetLoggerWithJSONFormat(t)
	})

	t.Run("CreatesLoggerWithDebugLevel", func(t *testing.T) {
		testGetLoggerWithDebugLevel(t)
	})

	t.Run("CreatesLoggerWithWarnLevel", func(t *testing.T) {
		testGetLoggerWithWarnLevel(t)
	})

	t.Run("CreatesLoggerWithErrorLevel", func(t *testing.T) {
		testGetLoggerWithErrorLevel(t)
	})

	t.Run("CreatesLoggerWithFatalLevel", func(t *testing.T) {
		testGetLoggerWithFatalLevel(t)
	})

	t.Run("CreatesLoggerWithPanicLevel", func(t *testing.T) {
		testGetLoggerWithPanicLevel(t)
	})

	t.Run("CreatesLoggerWithDefaultLevel", func(t *testing.T) {
		testGetLoggerWithDefaultLevel(t)
	})

	t.Run("LoggerOutputIsStderr", func(t *testing.T) {
		testGetLoggerOutputIsStderr(t)
	})

	t.Run("TextFormatterHasTimestampFormat", func(t *testing.T) {
		testGetLoggerTextFormatterTimestamp(t)
	})

	t.Run("JSONFormatterHasTimestampFormat", func(t *testing.T) {
		testGetLoggerJSONFormatterTimestamp(t)
	})
}

func TestAllParams(t *testing.T) {
	t.Parallel()

	t.Run("ExtractsAllConnectParams", func(t *testing.T) {
		testAllParamsConnectParams(t)
	})

	t.Run("ExtractsAllCollectionParams", func(t *testing.T) {
		testAllParamsCollectionParams(t)
	})

	t.Run("ExtractsAllOboGraphCollectionParams", func(t *testing.T) {
		testAllParamsOboGraphCollectionParams(t)
	})

	t.Run("HandlesIntegerConversion", func(t *testing.T) {
		testAllParamsIntegerConversion(t)
	})

	t.Run("HandlesBooleanValues", func(t *testing.T) {
		testAllParamsBooleanValues(t)
	})

	t.Run("ReturnsAllThreeParameterStructs", func(t *testing.T) {
		testAllParamsReturnsAllStructs(t)
	})

	t.Run("HandlesDefaultValues", func(t *testing.T) {
		testAllParamsWithDefaults(t)
	})
}

// TestRunServer documents the integration testing requirements for the RunServer function.
// This function is intentionally skipped as it requires external infrastructure.
// See package documentation above for detailed testing requirements and examples.
func TestRunServer(t *testing.T) {
	t.Skip(
		"RunServer requires integration testing with ArangoDB, NATS, and gRPC - see package documentation",
	)
}

// Test implementation functions for strainType

func testStrainTypeWithEmptyParams(t *testing.T) {
	t.Helper()
	option := strainType("general strain")
	serviceOpts := &aphgrpc.ServiceOptions{}

	option(serviceOpts)

	require.NotNil(t, serviceOpts.Params)
	require.Equal(t, "general strain", serviceOpts.Params["strain_term"])
}

func testStrainTypeWithExistingParams(t *testing.T) {
	t.Helper()
	option := strainType("general strain")
	serviceOpts := &aphgrpc.ServiceOptions{
		Params: map[string]string{
			"other_param": "other_value",
		},
	}

	option(serviceOpts)

	require.NotNil(t, serviceOpts.Params)
	require.Equal(t, "general strain", serviceOpts.Params["strain_term"])
	require.Equal(t, "other_value", serviceOpts.Params["other_param"])
}

func testStrainTypeOverwritesExisting(t *testing.T) {
	t.Helper()
	option := strainType("new strain type")
	serviceOpts := &aphgrpc.ServiceOptions{
		Params: map[string]string{
			"strain_term": "old strain type",
		},
	}

	option(serviceOpts)

	require.Equal(t, "new strain type", serviceOpts.Params["strain_term"])
}

func testStrainTypeWithEmptyString(t *testing.T) {
	t.Helper()
	option := strainType("")
	serviceOpts := &aphgrpc.ServiceOptions{}

	option(serviceOpts)

	require.NotNil(t, serviceOpts.Params)
	require.Equal(t, "", serviceOpts.Params["strain_term"])
}

// Test implementation functions for plasmidType

func testPlasmidTypeWithEmptyParams(t *testing.T) {
	t.Helper()
	option := plasmidType("cloning vector")
	serviceOpts := &aphgrpc.ServiceOptions{}

	option(serviceOpts)

	require.NotNil(t, serviceOpts.Params)
	require.Equal(t, "cloning vector", serviceOpts.Params["plasmid_term"])
}

func testPlasmidTypeWithExistingParams(t *testing.T) {
	t.Helper()
	option := plasmidType("cloning vector")
	serviceOpts := &aphgrpc.ServiceOptions{
		Params: map[string]string{
			"other_param": "other_value",
		},
	}

	option(serviceOpts)

	require.NotNil(t, serviceOpts.Params)
	require.Equal(t, "cloning vector", serviceOpts.Params["plasmid_term"])
	require.Equal(t, "other_value", serviceOpts.Params["other_param"])
}

func testPlasmidTypeOverwritesExisting(t *testing.T) {
	t.Helper()
	option := plasmidType("new plasmid type")
	serviceOpts := &aphgrpc.ServiceOptions{
		Params: map[string]string{
			"plasmid_term": "old plasmid type",
		},
	}

	option(serviceOpts)

	require.Equal(t, "new plasmid type", serviceOpts.Params["plasmid_term"])
}

func testPlasmidTypeWithEmptyString(t *testing.T) {
	t.Helper()
	option := plasmidType("")
	serviceOpts := &aphgrpc.ServiceOptions{}

	option(serviceOpts)

	require.NotNil(t, serviceOpts.Params)
	require.Equal(t, "", serviceOpts.Params["plasmid_term"])
}

// Test implementation functions for getLogger

func testGetLoggerWithTextFormat(t *testing.T) {
	t.Helper()
	ctx := createTestContext(map[string]string{
		flagLogFormat: logFormatText,
		flagLogLevel:  logLevelError,
	})

	logger := getLogger(ctx)

	require.NotNil(t, logger)
	require.NotNil(t, logger.Logger)
	_, ok := logger.Logger.Formatter.(*logrus.TextFormatter)
	require.True(t, ok, "logger should use TextFormatter")
}

func testGetLoggerWithJSONFormat(t *testing.T) {
	t.Helper()
	ctx := createTestContext(map[string]string{
		flagLogFormat: logFormatJSON,
		flagLogLevel:  logLevelError,
	})

	logger := getLogger(ctx)

	require.NotNil(t, logger)
	require.NotNil(t, logger.Logger)
	_, ok := logger.Logger.Formatter.(*logrus.JSONFormatter)
	require.True(t, ok, "logger should use JSONFormatter")
}

func testGetLoggerWithDebugLevel(t *testing.T) {
	t.Helper()
	ctx := createTestContext(map[string]string{
		flagLogFormat: logFormatText,
		flagLogLevel:  "debug",
	})

	logger := getLogger(ctx)

	require.NotNil(t, logger)
	require.Equal(t, logrus.DebugLevel, logger.Logger.Level)
}

func testGetLoggerWithWarnLevel(t *testing.T) {
	t.Helper()
	ctx := createTestContext(map[string]string{
		flagLogFormat: logFormatText,
		flagLogLevel:  "warn",
	})

	logger := getLogger(ctx)

	require.NotNil(t, logger)
	require.Equal(t, logrus.WarnLevel, logger.Logger.Level)
}

func testGetLoggerWithErrorLevel(t *testing.T) {
	t.Helper()
	ctx := createTestContext(map[string]string{
		flagLogFormat: logFormatText,
		flagLogLevel:  logLevelError,
	})

	logger := getLogger(ctx)

	require.NotNil(t, logger)
	require.Equal(t, logrus.ErrorLevel, logger.Logger.Level)
}

func testGetLoggerWithFatalLevel(t *testing.T) {
	t.Helper()
	ctx := createTestContext(map[string]string{
		flagLogFormat: logFormatText,
		flagLogLevel:  "fatal",
	})

	logger := getLogger(ctx)

	require.NotNil(t, logger)
	require.Equal(t, logrus.FatalLevel, logger.Logger.Level)
}

func testGetLoggerWithPanicLevel(t *testing.T) {
	t.Helper()
	ctx := createTestContext(map[string]string{
		flagLogFormat: logFormatText,
		flagLogLevel:  "panic",
	})

	logger := getLogger(ctx)

	require.NotNil(t, logger)
	require.Equal(t, logrus.PanicLevel, logger.Logger.Level)
}

func testGetLoggerWithDefaultLevel(t *testing.T) {
	t.Helper()
	ctx := createTestContext(map[string]string{
		flagLogFormat: logFormatText,
		flagLogLevel:  "info",
	})

	logger := getLogger(ctx)

	require.NotNil(t, logger)
	// When level is not recognized, it should use default (which is InfoLevel in logrus)
	require.Equal(t, logrus.InfoLevel, logger.Logger.Level)
}

func testGetLoggerOutputIsStderr(t *testing.T) {
	t.Helper()
	ctx := createTestContext(map[string]string{
		flagLogFormat: logFormatText,
		flagLogLevel:  logLevelError,
	})

	logger := getLogger(ctx)

	require.NotNil(t, logger)
	require.Equal(t, os.Stderr, logger.Logger.Out)
}

func testGetLoggerTextFormatterTimestamp(t *testing.T) {
	t.Helper()
	ctx := createTestContext(map[string]string{
		flagLogFormat: logFormatText,
		flagLogLevel:  logLevelError,
	})

	logger := getLogger(ctx)

	require.NotNil(t, logger)
	formatter, ok := logger.Logger.Formatter.(*logrus.TextFormatter)
	require.True(t, ok)
	require.Equal(t, "02/Jan/2006:15:04:05", formatter.TimestampFormat)
}

func testGetLoggerJSONFormatterTimestamp(t *testing.T) {
	t.Helper()
	ctx := createTestContext(map[string]string{
		flagLogFormat: logFormatJSON,
		flagLogLevel:  logLevelError,
	})

	logger := getLogger(ctx)

	require.NotNil(t, logger)
	formatter, ok := logger.Logger.Formatter.(*logrus.JSONFormatter)
	require.True(t, ok)
	require.Equal(t, "02/Jan/2006:15:04:05", formatter.TimestampFormat)
}

// Test implementation functions for allParams

func testAllParamsConnectParams(t *testing.T) {
	t.Helper()
	ctx := createTestContextWithFlags(map[string]any{
		"arangodb-user":     "testuser",
		"arangodb-pass":     "testpass",
		"arangodb-database": "testdb",
		"arangodb-host":     "localhost",
		flagArangodbPort:    "8529",
		flagIsSecure:        true,
	})

	connP, _, _ := allParams(ctx)

	require.NotNil(t, connP)
	require.Equal(t, "testuser", connP.User)
	require.Equal(t, "testpass", connP.Pass)
	require.Equal(t, "testdb", connP.Database)
	require.Equal(t, "localhost", connP.Host)
	require.Equal(t, 8529, connP.Port)
	require.True(t, connP.Istls)
}

func testAllParamsCollectionParams(t *testing.T) {
	t.Helper()
	ctx := createTestContextWithFlags(map[string]any{
		"stock-collection":               "stock",
		"stockprop-collection":           "stockprop",
		"stock-key-generator-collection": "stock_key_gen",
		"stock-type-edge":                "stock_type",
		"parent-strain-edge":             "parent_strain",
		"stockproptype-graph":            "stockprop_type",
		"strain2parent-graph":            "strain2parent",
		"strain-ontology":                "dicty_strain_property",
		"plasmid-ontology":               "plasmid_keywords",
		"keyoffset":                      370000,
		"stock-term-edge":                "stock_term",
		"stockonto-graph":                "stockonto",
	})

	_, collP, _ := allParams(ctx)

	require.NotNil(t, collP)
	require.Equal(t, "stock", collP.Stock)
	require.Equal(t, "stockprop", collP.StockProp)
	require.Equal(t, "stock_key_gen", collP.StockKeyGenerator)
	require.Equal(t, "stock_type", collP.StockType)
	require.Equal(t, "parent_strain", collP.ParentStrain)
	require.Equal(t, "stockprop_type", collP.StockPropTypeGraph)
	require.Equal(t, "strain2parent", collP.Strain2ParentGraph)
	require.Equal(t, "dicty_strain_property", collP.StrainOntology)
	require.Equal(t, "plasmid_keywords", collP.PlasmidOntology)
	require.Equal(t, 370000, collP.KeyOffset)
	require.Equal(t, "stock_term", collP.StockTerm)
	require.Equal(t, "stockonto", collP.StockOntoGraph)
}

func testAllParamsOboGraphCollectionParams(t *testing.T) {
	t.Helper()
	ctx := createTestContextWithFlags(map[string]any{
		"cv-collection":   "cvterm",
		"obograph":        "obograph",
		"rel-collection":  "cvterm_relationship",
		"term-collection": "cvterm",
	})

	_, _, ontoP := allParams(ctx)

	require.NotNil(t, ontoP)
	require.Equal(t, "cvterm", ontoP.GraphInfo)
	require.Equal(t, "obograph", ontoP.OboGraph)
	require.Equal(t, "cvterm_relationship", ontoP.Relationship)
	require.Equal(t, "cvterm", ontoP.Term)
}

func testAllParamsIntegerConversion(t *testing.T) {
	t.Helper()
	ctx := createTestContextWithFlags(map[string]any{
		flagArangodbPort: "9999",
		"keyoffset":      500000,
	})

	connP, collP, _ := allParams(ctx)

	require.NotNil(t, connP)
	require.Equal(t, 9999, connP.Port)
	require.NotNil(t, collP)
	require.Equal(t, 500000, collP.KeyOffset)
}

func testAllParamsBooleanValues(t *testing.T) {
	t.Helper()
	ctx := createTestContextWithFlags(map[string]any{
		flagIsSecure: false,
	})

	connP, _, _ := allParams(ctx)

	require.NotNil(t, connP)
	require.False(t, connP.Istls)
}

func testAllParamsReturnsAllStructs(t *testing.T) {
	t.Helper()
	ctx := createTestContextWithFlags(map[string]any{
		"arangodb-user":     "user",
		"arangodb-pass":     "pass",
		"arangodb-database": "db",
		"arangodb-host":     "host",
		flagArangodbPort:    "8529",
		flagIsSecure:        true,
		"stock-collection":  "stock",
	})

	connP, collP, ontoP := allParams(ctx)

	require.NotNil(t, connP, "ConnectParams should not be nil")
	require.NotNil(t, collP, "CollectionParams should not be nil")
	require.NotNil(t, ontoP, "OboGraph CollectionParams should not be nil")
}

func testAllParamsWithDefaults(t *testing.T) {
	t.Helper()
	// Create context with minimal flags - most values will be empty/default
	ctx := createTestContextWithFlags(map[string]any{})

	connP, collP, ontoP := allParams(ctx)

	// Should still return all three structs, even with empty values
	require.NotNil(t, connP)
	require.NotNil(t, collP)
	require.NotNil(t, ontoP)

	// Port conversion from empty string should result in 0
	require.Equal(t, 0, connP.Port)
	require.Equal(t, 0, collP.KeyOffset)
}

// Helper functions

func createTestContext(globalFlags map[string]string) *cli.Context {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", flag.ContinueOnError)

	// Add global flags
	for key, value := range globalFlags {
		set.String(key, value, "")
	}

	// Parse the flags
	for key, value := range globalFlags {
		_ = set.Set(key, value)
	}

	parentCtx := cli.NewContext(app, set, nil)
	return parentCtx
}

func createTestContextWithFlags(flags map[string]any) *cli.Context {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", flag.ContinueOnError)

	// Add and set flags based on their type
	for key, value := range flags {
		switch v := value.(type) {
		case string:
			set.String(key, v, "")
			_ = set.Set(key, v)
		case int:
			set.Int(key, v, "")
		case bool:
			set.Bool(key, v, "")
		}
	}

	return cli.NewContext(app, set, nil)
}

func (s stubGatewayStockServer) CountStocks(
	_ context.Context,
	_ *empty.Empty,
) (*stock.StockCounts, error) {
	if s.countCalls != nil {
		s.countCalls.Add(1)
	}
	return &stock.StockCounts{
		Strains: &stock.StockEntityCounts{
			Total: 8,
			Subtypes: []*stock.StockSubtypeCount{
				{Name: "regular", Count: 5},
				{Name: "gwdi", Count: 2},
				{Name: "bacterial", Count: 1},
			},
		},
		Plasmids: &stock.StockEntityCounts{
			Total: 3,
			Subtypes: []*stock.StockSubtypeCount{
				{Name: "regular", Count: 1},
				{Name: "goldenbraid", Count: 2},
			},
		},
	}, nil
}

func (stubGatewayStockServer) RemoveStock(
	_ context.Context,
	_ *stock.StockId,
) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}

func TestMultiplexedServerServesGRPCAndHTTPOnSamePort(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(err)
	addr := lis.Addr().String()

	store := cache.NewMemoryStore()
	t.Cleanup(func() { _ = store.Close() })

	var calls atomic.Int32
	grpcS := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			middleware.StockCountsCacheInterceptor(
				store,
				middleware.DefaultStockCountsCacheTTL,
			),
		),
	)
	svc := stubGatewayStockServer{countCalls: &calls}
	stock.RegisterStockServiceServer(grpcS, svc)

	httpSRes := F.Pipe1(
		newMultiplexedServer(gatewayInput{
			ctx:      context.Background(),
			grpcS:    grpcS,
			endpoint: addr,
		}),
		ioeutils.ToEither[error, *http.Server],
	)
	assert.True(E.IsRight(httpSRes))
	httpS := F.Pipe1(
		httpSRes,
		E.GetOrElse(F.Constant1[error]((*http.Server)(nil))),
	)
	assert.NotNil(httpS)

	go func() {
		_ = httpS.Serve(lis)
	}()
	t.Cleanup(func() {
		_ = httpS.Close()
		grpcS.Stop()
	})

	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	assert.NoError(err)
	t.Cleanup(func() {
		_ = conn.Close()
	})

	client := stock.NewStockServiceClient(conn)
	grpcResp, err := client.CountStocks(context.Background(), &empty.Empty{})
	assert.NoError(err)
	assert.Equal(int64(8), grpcResp.GetStrains().GetTotal())
	assert.Equal(int64(3), grpcResp.GetPlasmids().GetTotal())
	assert.Equal(int32(1), calls.Load())

	// HTTP gateway request hits the gRPC cache interceptor and does not re-invoke CountStocks.
	httpResp, err := http.Get("http://" + addr + "/v1/stocks/counts")
	assert.NoError(err)
	defer func() { _ = httpResp.Body.Close() }()
	assert.Equal(http.StatusOK, httpResp.StatusCode)
	body, err := io.ReadAll(httpResp.Body)
	assert.NoError(err)
	assert.Equal(int32(1), calls.Load())
	assert.JSONEq(
		`{
		  "strains": {
		    "total": "8",
		    "subtypes": [
		      {"name": "regular", "count": "5"},
		      {"name": "gwdi", "count": "2"},
		      {"name": "bacterial", "count": "1"}
		    ]
		  },
		  "plasmids": {
		    "total": "3",
		    "subtypes": [
		      {"name": "regular", "count": "1"},
		      {"name": "goldenbraid", "count": "2"}
		    ]
		  }
		}`,
		string(body),
	)

	// Mutation RPC invalidates the cache so the next HTTP request recomputes.
	_, err = client.RemoveStock(context.Background(), &stock.StockId{Id: "DBS1"})
	assert.NoError(err)

	httpResp2, err := http.Get("http://" + addr + "/v1/stocks/counts")
	assert.NoError(err)
	defer func() { _ = httpResp2.Body.Close() }()
	assert.Equal(http.StatusOK, httpResp2.StatusCode)
	assert.Equal(int32(2), calls.Load())

	unmappedResp, err := http.Get("http://" + addr + "/v1/stocks/strains")
	assert.NoError(err)
	defer func() { _ = unmappedResp.Body.Close() }()
	assert.Equal(http.StatusNotFound, unmappedResp.StatusCode)
}

func TestOpenCacheStore(t *testing.T) {
	t.Parallel()

	t.Run("DefaultsToMemoryStoreWhenRedisFlagsAreEmpty", func(t *testing.T) {
		t.Parallel()
		ctx := createTestContextWithFlags(map[string]any{})
		res := F.Pipe1(
			openCacheStore(serverState{cliCtx: ctx}),
			ioeutils.ToEither[error, cache.Store],
		)
		require.True(t, E.IsRight(res))
		store := F.Pipe1(res, E.GetOrElse(F.Constant1[error](cache.Store(nil))))
		require.IsType(t, &cache.MemoryStore{}, store)
		_ = store.Close()
	})

	t.Run("UsesRedisStoreWhenRedisHostAndPortAreProvided", func(t *testing.T) {
		t.Parallel()
		mr, err := miniredis.Run()
		require.NoError(t, err)
		t.Cleanup(mr.Close)

		host, port, err := net.SplitHostPort(mr.Addr())
		require.NoError(t, err)

		ctx := createTestContextWithFlags(map[string]any{
			flagRedisHost: host,
			flagRedisPort: port,
		})
		res := F.Pipe1(
			openCacheStore(serverState{cliCtx: ctx}),
			ioeutils.ToEither[error, cache.Store],
		)
		require.True(t, E.IsRight(res))
		store := F.Pipe1(res, E.GetOrElse(F.Constant1[error](cache.Store(nil))))
		require.IsType(t, &cache.RedisStore{}, store)
		_ = store.Close()
	})

	t.Run("FailsWhenConfiguredRedisIsUnreachable", func(t *testing.T) {
		t.Parallel()
		ctx := createTestContextWithFlags(map[string]any{
			flagRedisHost: "127.0.0.1",
			flagRedisPort: "1",
		})
		res := F.Pipe1(
			openCacheStore(serverState{cliCtx: ctx}),
			ioeutils.ToEither[error, cache.Store],
		)
		require.True(t, E.IsLeft(res))
	})
}

func TestServerPipelineAndLoomStyle(t *testing.T) {
	t.Parallel()
	cfg := pipelinecheck.Config{
		Roots: []string{"."},
		IsEntrypoint: func(name string) bool {
			return strings.HasPrefix(name, "Run") || strings.HasPrefix(name, "run")
		},
		RequirePointFreeSeed:      true,
		RequirePointFreeBranching: true,
		RequireTrivialDelegate:    true,
	}
	pipelinecheck.Require(t, cfg)
	pipelinecheck.RequireBareIf(t, cfg)
	pipelinecheck.RequireNoIfErrInTryCatch(t, cfg)
	pipelinecheck.RequireNoNonRawTryCatchCallback(t, cfg)
	pipelinecheck.RequireSafePrintf(t, cfg)
	pipelinecheck.RequireNoRedundantFold(t, cfg)
	pipelinecheck.RequireNoHandRolledFileJoinInArrayMap(t, cfg)
	pipelinecheck.RequireNoImperativeLoop(t, cfg)
	pipelinecheck.RequireNoImperativeBranch(t, cfg)
	pipelinecheck.RequireNoDirectApply(t, cfg)
	loomcheck.Require(t, loomcheck.Config{
		Roots:               []string{"."},
		StrictAllPredicates: true,
	})
}

func TestLoopbackDialEndpoint(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	//nolint:gosec // G102: the test intentionally binds a wildcard listener.
	wildcardLis, err := net.Listen("tcp", ":0")
	assert.NoError(err)
	t.Cleanup(func() { _ = wildcardLis.Close() })
	_, port, err := net.SplitHostPort(wildcardLis.Addr().String())
	assert.NoError(err)

	endpoint, err := loopbackDialEndpoint(wildcardLis)
	assert.NoError(err)
	assert.Equal(net.JoinHostPort("127.0.0.1", port), endpoint)

	loopbackLis, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(err)
	t.Cleanup(func() { _ = loopbackLis.Close() })
	loopbackEndpoint, err := loopbackDialEndpoint(loopbackLis)
	assert.NoError(err)
	assert.Equal(loopbackLis.Addr().String(), loopbackEndpoint)
}

func TestGatewayWithWildcardListenerServesHTTPParity(t *testing.T) {
	t.Parallel()
	assert := require.New(t)

	//nolint:gosec // G102: the test intentionally binds a wildcard listener.
	lis, err := net.Listen("tcp", ":0")
	assert.NoError(err)

	store := cache.NewMemoryStore()
	t.Cleanup(func() { _ = store.Close() })

	var calls atomic.Int32
	grpcS := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			middleware.StockCountsCacheInterceptor(
				store,
				middleware.DefaultStockCountsCacheTTL,
			),
		),
	)
	stock.RegisterStockServiceServer(grpcS, stubGatewayStockServer{countCalls: &calls})

	httpSRes := F.Pipe1(
		buildGatewayServerFromState(serverState{grpcS: grpcS, lis: lis}),
		ioeutils.ToEither[error, *http.Server],
	)
	assert.True(E.IsRight(httpSRes))
	httpS := F.Pipe1(
		httpSRes,
		E.GetOrElse(F.Constant1[error]((*http.Server)(nil))),
	)
	assert.NotNil(httpS)

	go func() {
		_ = httpS.Serve(lis)
	}()
	t.Cleanup(func() {
		_ = httpS.Close()
		grpcS.Stop()
	})

	_, port, err := net.SplitHostPort(lis.Addr().String())
	assert.NoError(err)
	dialAddr := net.JoinHostPort("127.0.0.1", port)

	conn, err := grpc.NewClient(
		dialAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	assert.NoError(err)
	t.Cleanup(func() {
		_ = conn.Close()
	})

	client := stock.NewStockServiceClient(conn)
	grpcResp, err := client.CountStocks(context.Background(), &empty.Empty{})
	assert.NoError(err)
	assert.Equal(int64(8), grpcResp.GetStrains().GetTotal())
	assert.Equal(int32(1), calls.Load())

	// The HTTP gateway must reach the same cache entry through the
	// loopback endpoint even when the listener binds a wildcard address.
	httpResp, err := http.Get("http://" + dialAddr + "/v1/stocks/counts")
	assert.NoError(err)
	defer func() { _ = httpResp.Body.Close() }()
	assert.Equal(http.StatusOK, httpResp.StatusCode)
	assert.Equal(int32(1), calls.Load())
}
