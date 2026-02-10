package server

import (
	"flag"
	"os"
	"testing"

	"github.com/dictyBase/aphgrpc"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli"
)

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
		"log-format": "text",
		"log-level":  "error",
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
		"log-format": "json",
		"log-level":  "error",
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
		"log-format": "text",
		"log-level":  "debug",
	})

	logger := getLogger(ctx)

	require.NotNil(t, logger)
	require.Equal(t, logrus.DebugLevel, logger.Logger.Level)
}

func testGetLoggerWithWarnLevel(t *testing.T) {
	t.Helper()
	ctx := createTestContext(map[string]string{
		"log-format": "text",
		"log-level":  "warn",
	})

	logger := getLogger(ctx)

	require.NotNil(t, logger)
	require.Equal(t, logrus.WarnLevel, logger.Logger.Level)
}

func testGetLoggerWithErrorLevel(t *testing.T) {
	t.Helper()
	ctx := createTestContext(map[string]string{
		"log-format": "text",
		"log-level":  "error",
	})

	logger := getLogger(ctx)

	require.NotNil(t, logger)
	require.Equal(t, logrus.ErrorLevel, logger.Logger.Level)
}

func testGetLoggerWithFatalLevel(t *testing.T) {
	t.Helper()
	ctx := createTestContext(map[string]string{
		"log-format": "text",
		"log-level":  "fatal",
	})

	logger := getLogger(ctx)

	require.NotNil(t, logger)
	require.Equal(t, logrus.FatalLevel, logger.Logger.Level)
}

func testGetLoggerWithPanicLevel(t *testing.T) {
	t.Helper()
	ctx := createTestContext(map[string]string{
		"log-format": "text",
		"log-level":  "panic",
	})

	logger := getLogger(ctx)

	require.NotNil(t, logger)
	require.Equal(t, logrus.PanicLevel, logger.Logger.Level)
}

func testGetLoggerWithDefaultLevel(t *testing.T) {
	t.Helper()
	ctx := createTestContext(map[string]string{
		"log-format": "text",
		"log-level":  "info",
	})

	logger := getLogger(ctx)

	require.NotNil(t, logger)
	// When level is not recognized, it should use default (which is InfoLevel in logrus)
	require.Equal(t, logrus.InfoLevel, logger.Logger.Level)
}

func testGetLoggerOutputIsStderr(t *testing.T) {
	t.Helper()
	ctx := createTestContext(map[string]string{
		"log-format": "text",
		"log-level":  "error",
	})

	logger := getLogger(ctx)

	require.NotNil(t, logger)
	require.Equal(t, os.Stderr, logger.Logger.Out)
}

func testGetLoggerTextFormatterTimestamp(t *testing.T) {
	t.Helper()
	ctx := createTestContext(map[string]string{
		"log-format": "text",
		"log-level":  "error",
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
		"log-format": "json",
		"log-level":  "error",
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
		"arangodb-port":     "8529",
		"is-secure":         true,
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
		"arangodb-port": "9999",
		"keyoffset":     500000,
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
		"is-secure": false,
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
		"arangodb-port":     "8529",
		"is-secure":         true,
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
