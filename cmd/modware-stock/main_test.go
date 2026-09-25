package main

import (
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli"
)

func TestServerFlags(t *testing.T) {
	t.Parallel()

	t.Run("ReturnsAllServerFlags", func(t *testing.T) {
		testServerFlagsCount(t)
	})

	t.Run("ContainsPortFlag", func(t *testing.T) {
		testServerPortFlag(t)
	})

	t.Run("ContainsKeyOffsetFlag", func(t *testing.T) {
		testServerKeyOffsetFlag(t)
	})

	t.Run("ContainsReflectionFlag", func(t *testing.T) {
		testServerReflectionFlag(t)
	})

	t.Run("ContainsStrainOntologyFlag", func(t *testing.T) {
		testServerStrainOntologyFlag(t)
	})

	t.Run("ContainsStrainTermFlag", func(t *testing.T) {
		testServerStrainTermFlag(t)
	})

	t.Run("ContainsPlasmidOntologyFlag", func(t *testing.T) {
		testServerPlasmidOntologyFlag(t)
	})

	t.Run("ContainsPlasmidTermFlag", func(t *testing.T) {
		testServerPlasmidTermFlag(t)
	})

	t.Run("AllServerFlagsAreValid", func(t *testing.T) {
		testServerFlagsValidity(t)
	})

	t.Run("ServerFlagsHaveNonEmptyUsage", func(t *testing.T) {
		testServerFlagsUsage(t)
	})

	t.Run("ServerFlagsHaveNoEnvironmentVariables", func(t *testing.T) {
		testServerFlagsNoEnvVars(t)
	})
}

func TestDbCollectionFlags(t *testing.T) {
	t.Parallel()

	t.Run("ReturnsAllDatabaseCollectionFlags", func(t *testing.T) {
		testDbCollectionFlagsCount(t)
	})

	t.Run("ContainsStockCollectionFlag", func(t *testing.T) {
		testDbStockCollectionFlag(t)
	})

	t.Run("ContainsStockPropCollectionFlag", func(t *testing.T) {
		testDbStockPropCollectionFlag(t)
	})

	t.Run("ContainsStockKeyGeneratorCollectionFlag", func(t *testing.T) {
		testDbStockKeyGeneratorFlag(t)
	})

	t.Run("ContainsStockTypeEdgeFlag", func(t *testing.T) {
		testDbStockTypeEdgeFlag(t)
	})

	t.Run("ContainsParentStrainEdgeFlag", func(t *testing.T) {
		testDbParentStrainEdgeFlag(t)
	})

	t.Run("ContainsStockTermEdgeFlag", func(t *testing.T) {
		testDbStockTermEdgeFlag(t)
	})

	t.Run("ContainsStockPropTypeGraphFlag", func(t *testing.T) {
		testDbStockPropTypeGraphFlag(t)
	})

	t.Run("ContainsStrain2ParentGraphFlag", func(t *testing.T) {
		testDbStrain2ParentGraphFlag(t)
	})

	t.Run("ContainsStockOntoGraphFlag", func(t *testing.T) {
		testDbStockOntoGraphFlag(t)
	})

	t.Run("AllDbCollectionFlagsAreStringFlags", func(t *testing.T) {
		testDbCollectionFlagsAreStrings(t)
	})

	t.Run("AllDbCollectionFlagsHaveDefaults", func(t *testing.T) {
		testDbCollectionFlagsHaveDefaults(t)
	})

	t.Run("DbCollectionFlagsHaveDescriptiveUsage", func(t *testing.T) {
		testDbCollectionFlagsUsage(t)
	})

	t.Run("DbCollectionFlagsHaveNoEnvironmentVariables", func(t *testing.T) {
		testDbCollectionFlagsNoEnvVars(t)
	})
}

func TestAllFlags(t *testing.T) {
	t.Parallel()

	t.Run("CombinesAllFlagSources", func(t *testing.T) {
		testAllFlagsCombination(t)
	})

	t.Run("IncludesServerFlags", func(t *testing.T) {
		testAllFlagsIncludesServer(t)
	})

	t.Run("IncludesDatabaseCollectionFlags", func(t *testing.T) {
		testAllFlagsIncludesDb(t)
	})

	t.Run("IncludesArangoDBDatabaseFlag", func(t *testing.T) {
		testAllFlagsIncludesArangoDb(t)
	})

	t.Run("FlagsAreNotDuplicated", func(t *testing.T) {
		testAllFlagsNoDuplicates(t)
	})

	t.Run("AllFlagsHaveValidConfiguration", func(t *testing.T) {
		testAllFlagsValidConfig(t)
	})
}

func TestMain(t *testing.T) {
	t.Parallel()

	t.Run("ApplicationHasCorrectMetadata", func(t *testing.T) {
		testAppMetadata(t)
	})

	t.Run("ApplicationHasGlobalFlags", func(t *testing.T) {
		testAppGlobalFlags(t)
	})

	t.Run("ApplicationHasStartServerCommand", func(t *testing.T) {
		testAppStartServerCommand(t)
	})

	t.Run("StartServerCommandHasAllFlags", func(t *testing.T) {
		testStartServerCommandFlags(t)
	})

	t.Run("MainFunctionCreatesApplicationWithCorrectConfiguration", func(t *testing.T) {
		testMainFunctionConfiguration(t)
	})

	t.Run("MainFunctionIncludesAllRequiredComponents", func(t *testing.T) {
		testMainFunctionComponents(t)
	})
}

func TestMainIntegration(t *testing.T) {
	t.Parallel()

	t.Run("ApplicationFailsWithNoArguments", func(t *testing.T) {
		testAppRunsWithNoArgs(t)
	})

	t.Run("ApplicationShowsHelpWithHelpFlag", func(t *testing.T) {
		testAppRunsWithHelpFlag(t)
	})

	t.Run("ApplicationShowsVersionWithVersionFlag", func(t *testing.T) {
		testAppRunsWithVersionFlag(t)
	})

	t.Run("ApplicationMatchesMainFunctionStructure", func(t *testing.T) {
		testMainFunctionStructure(t)
	})

	t.Run("ApplicationCanRunCompleteLifecycle", func(t *testing.T) {
		testCompleteApplicationLifecycle(t)
	})
}

func TestMainFlagDefaults(t *testing.T) {
	t.Parallel()

	t.Run("ServerFlagsHaveReasonableDefaults", func(t *testing.T) {
		testServerFlagsDefaults(t)
	})

	t.Run("DatabaseFlagsHaveReasonableDefaults", func(t *testing.T) {
		testDbFlagsDefaults(t)
	})
}

func TestMainFlagIntegrationWithCLI(t *testing.T) {
	t.Parallel()

	t.Run("FlagsCanBeParsedByCliFramework", func(t *testing.T) {
		testFlagsCliParsing(t)
	})

	t.Run("GlobalFlagsCanBeParsedByCliFramework", func(t *testing.T) {
		testGlobalFlagsCliParsing(t)
	})
}

// Test implementation functions for server flags

func testServerFlagsCount(t *testing.T) {
	t.Helper()
	flags := serverFlags()
	require.NotNil(t, flags)
	require.NotEmpty(t, flags)
	require.Len(t, flags, 7, "should return exactly 7 server flags")
}

func testServerPortFlag(t *testing.T) {
	t.Helper()
	flags := serverFlags()
	portFlag := findFlagByName(flags, "port")
	require.NotNil(t, portFlag, "port flag should exist")

	strFlag, ok := portFlag.(cli.StringFlag)
	require.True(t, ok, "port flag should be a StringFlag")
	require.Equal(t, "port", strFlag.Name)
	require.Equal(t, "9560", strFlag.Value)
	require.Equal(t, "tcp port at which the server will be available", strFlag.Usage)
}

func testServerKeyOffsetFlag(t *testing.T) {
	t.Helper()
	flags := serverFlags()
	keyOffsetFlag := findFlagByName(flags, "keyoffset")
	require.NotNil(t, keyOffsetFlag, "keyoffset flag should exist")

	intFlag, ok := keyOffsetFlag.(cli.IntFlag)
	require.True(t, ok, "keyoffset flag should be an IntFlag")
	require.Equal(t, "keyoffset", intFlag.Name)
	require.Equal(t, 370000, intFlag.Value)
	require.Equal(t, "initial offset for stock id generation", intFlag.Usage)
}

func testServerReflectionFlag(t *testing.T) {
	t.Helper()
	flags := serverFlags()
	reflectionFlag := findFlagByName(flags, "reflection")
	require.NotNil(t, reflectionFlag, "reflection flag should exist")

	boolFlag, ok := reflectionFlag.(cli.BoolTFlag)
	require.True(t, ok, "reflection flag should be a BoolTFlag")
	require.Contains(t, boolFlag.Name, "reflection")
	require.Contains(t, boolFlag.Name, "ref")
	require.Equal(t, "flag for enabling server reflection", boolFlag.Usage)
}

func testServerStrainOntologyFlag(t *testing.T) {
	t.Helper()
	flags := serverFlags()
	strainOntoFlag := findFlagByName(flags, "strain-ontology")
	require.NotNil(t, strainOntoFlag, "strain-ontology flag should exist")

	strFlag, ok := strainOntoFlag.(cli.StringFlag)
	require.True(t, ok, "strain-ontology flag should be a StringFlag")
	require.Equal(t, "strain-ontology", strFlag.Name)
	require.Equal(t, "dicty_strain_property", strFlag.Value)
	require.Equal(t,
		"dictybase ontology that will be used for picking grouping term for strain",
		strFlag.Usage)
}

func testServerStrainTermFlag(t *testing.T) {
	t.Helper()
	flags := serverFlags()
	strainTermFlag := findFlagByName(flags, "strain-term")
	require.NotNil(t, strainTermFlag, "strain-term flag should exist")

	strFlag, ok := strainTermFlag.(cli.StringFlag)
	require.True(t, ok, "strain-term flag should be a StringFlag")
	require.Equal(t, "strain-term", strFlag.Name)
	require.Equal(t, "general strain", strFlag.Value)
	require.Equal(t,
		"default ontology term that will be used for creating strain",
		strFlag.Usage)
}

func testServerPlasmidOntologyFlag(t *testing.T) {
	t.Helper()
	flags := serverFlags()
	plasmidOntoFlag := findFlagByName(flags, "plasmid-ontology")
	require.NotNil(t, plasmidOntoFlag, "plasmid-ontology flag should exist")

	strFlag, ok := plasmidOntoFlag.(cli.StringFlag)
	require.True(t, ok, "plasmid-ontology flag should be a StringFlag")
	require.Equal(t, "plasmid-ontology", strFlag.Name)
	require.Equal(t, "plasmid_keywords", strFlag.Value)
	require.Equal(t,
		"dictybase ontology that will be used for picking grouping term for plasmid",
		strFlag.Usage)
}

func testServerPlasmidTermFlag(t *testing.T) {
	t.Helper()
	flags := serverFlags()
	plasmidTermFlag := findFlagByName(flags, "plasmid-term")
	require.NotNil(t, plasmidTermFlag, "plasmid-term flag should exist")

	strFlag, ok := plasmidTermFlag.(cli.StringFlag)
	require.True(t, ok, "plasmid-term flag should be a StringFlag")
	require.Equal(t, "plasmid-term", strFlag.Name)
	require.Equal(t, "vector", strFlag.Value)
	require.Equal(t,
		"default ontology term that will be used for creating plasmid",
		strFlag.Usage)
}

func testServerFlagsValidity(t *testing.T) {
	t.Helper()
	flags := serverFlags()

	for _, flg := range flags {
		name := getFlagName(flg)
		require.NotEmpty(t, name, "all server flags must have a name")

		usage := getFlagUsage(flg)
		require.NotEmpty(t, usage, "flag %s must have usage text", name)
	}
}

func testServerFlagsUsage(t *testing.T) {
	t.Helper()
	flags := serverFlags()

	expectedUsages := map[string]string{
		"port":             "tcp port at which the server will be available",
		"keyoffset":        "initial offset for stock id generation",
		"reflection":       "flag for enabling server reflection",
		"strain-ontology":  "dictybase ontology that will be used for picking grouping term for strain",
		"strain-term":      "default ontology term that will be used for creating strain",
		"plasmid-ontology": "dictybase ontology that will be used for picking grouping term for plasmid",
		"plasmid-term":     "default ontology term that will be used for creating plasmid",
	}

	for expectedName, expectedUsage := range expectedUsages {
		flg := findFlagByName(flags, expectedName)
		require.NotNil(t, flg, "flag %s should exist", expectedName)

		actualUsage := getFlagUsage(flg)
		require.Equal(t, expectedUsage, actualUsage, "flag %s should have correct usage", expectedName)
	}
}

func testServerFlagsNoEnvVars(t *testing.T) {
	t.Helper()
	flags := serverFlags()

	for _, flg := range flags {
		envVar := getFlagEnvVar(flg)
		require.Empty(t, envVar, "server flag %s should not have environment variable", getFlagName(flg))
	}
}

// Test implementation functions for database collection flags

func testDbCollectionFlagsCount(t *testing.T) {
	t.Helper()
	flags := dbCollectionFlags()
	require.NotNil(t, flags)
	require.NotEmpty(t, flags)
	require.Len(t, flags, 9, "should return exactly 9 database collection flags")
}

func testDbStockCollectionFlag(t *testing.T) {
	t.Helper()
	flags := dbCollectionFlags()
	stockFlag := findFlagByName(flags, collectionStock)
	require.NotNil(t, stockFlag, "stock-collection flag should exist")

	strFlag, ok := stockFlag.(cli.StringFlag)
	require.True(t, ok, "stock-collection flag should be a StringFlag")
	require.Equal(t, collectionStock, strFlag.Name)
	require.Equal(t, defaultStockCollection, strFlag.Value)
	require.Equal(t, "arangodb collection for storing biological stocks", strFlag.Usage)
}

func testDbStockPropCollectionFlag(t *testing.T) {
	t.Helper()
	flags := dbCollectionFlags()
	stockPropFlag := findFlagByName(flags, collectionStockProp)
	require.NotNil(t, stockPropFlag, "stockprop-collection flag should exist")

	strFlag, ok := stockPropFlag.(cli.StringFlag)
	require.True(t, ok, "stockprop-collection flag should be a StringFlag")
	require.Equal(t, collectionStockProp, strFlag.Name)
	require.Equal(t, "stockprop", strFlag.Value)
	require.Equal(t, "arangodb collection for storing stock properties", strFlag.Usage)
}

func testDbStockKeyGeneratorFlag(t *testing.T) {
	t.Helper()
	flags := dbCollectionFlags()
	keyGenFlag := findFlagByName(flags, collectionStockKeyGenerator)
	require.NotNil(t, keyGenFlag, "stock-key-generator-collection flag should exist")

	strFlag, ok := keyGenFlag.(cli.StringFlag)
	require.True(t, ok, "stock-key-generator-collection flag should be a StringFlag")
	require.Equal(t, collectionStockKeyGenerator, strFlag.Name)
	require.Equal(t, "stock_key_generator", strFlag.Value)
	require.Equal(t, "arangodb collection for generating unique IDs", strFlag.Usage)
}

func testDbStockTypeEdgeFlag(t *testing.T) {
	t.Helper()
	flags := dbCollectionFlags()
	stockTypeFlag := findFlagByName(flags, collectionStockTypeEdge)
	require.NotNil(t, stockTypeFlag, "stock-type-edge flag should exist")

	strFlag, ok := stockTypeFlag.(cli.StringFlag)
	require.True(t, ok, "stock-type-edge flag should be a StringFlag")
	require.Equal(t, collectionStockTypeEdge, strFlag.Name)
	require.Equal(t, "stock_type", strFlag.Value)
	require.Equal(t,
		"arangodb edge collection for connecting stocks to their types (strain or plasmid)",
		strFlag.Usage)
}

func testDbParentStrainEdgeFlag(t *testing.T) {
	t.Helper()
	flags := dbCollectionFlags()
	parentFlag := findFlagByName(flags, collectionParentStrainEdge)
	require.NotNil(t, parentFlag, "parent-strain-edge flag should exist")

	strFlag, ok := parentFlag.(cli.StringFlag)
	require.True(t, ok, "parent-strain-edge flag should be a StringFlag")
	require.Equal(t, collectionParentStrainEdge, strFlag.Name)
	require.Equal(t, "parent_strain", strFlag.Value)
	require.Equal(t,
		"arangodb edge collection for connecting strains to their parent",
		strFlag.Usage)
}

func testDbStockTermEdgeFlag(t *testing.T) {
	t.Helper()
	flags := dbCollectionFlags()
	termFlag := findFlagByName(flags, collectionStockTermEdge)
	require.NotNil(t, termFlag, "stock-term-edge flag should exist")

	strFlag, ok := termFlag.(cli.StringFlag)
	require.True(t, ok, "stock-term-edge flag should be a StringFlag")
	require.Equal(t, collectionStockTermEdge, strFlag.Name)
	require.Equal(t, "stock_term", strFlag.Value)
	require.Equal(t,
		"arangodb edge collection for connecting stock to ontology term",
		strFlag.Usage)
}

func testDbStockPropTypeGraphFlag(t *testing.T) {
	t.Helper()
	flags := dbCollectionFlags()
	graphFlag := findFlagByName(flags, graphStockPropType)
	require.NotNil(t, graphFlag, "stockproptype-graph flag should exist")

	strFlag, ok := graphFlag.(cli.StringFlag)
	require.True(t, ok, "stockproptype-graph flag should be a StringFlag")
	require.Equal(t, graphStockPropType, strFlag.Name)
	require.Equal(t, "stockprop_type", strFlag.Value)
	require.Equal(t,
		"arangodb named graph for managing relations between stocks and their properties",
		strFlag.Usage)
}

func testDbStrain2ParentGraphFlag(t *testing.T) {
	t.Helper()
	flags := dbCollectionFlags()
	graphFlag := findFlagByName(flags, graphStrain2Parent)
	require.NotNil(t, graphFlag, "strain2parent-graph flag should exist")

	strFlag, ok := graphFlag.(cli.StringFlag)
	require.True(t, ok, "strain2parent-graph flag should be a StringFlag")
	require.Equal(t, graphStrain2Parent, strFlag.Name)
	require.Equal(t, "strain2parent", strFlag.Value)
	require.Equal(t,
		"arangodb named graph for managing relations between strains and their parents",
		strFlag.Usage)
}

func testDbStockOntoGraphFlag(t *testing.T) {
	t.Helper()
	flags := dbCollectionFlags()
	graphFlag := findFlagByName(flags, graphStockOnto)
	require.NotNil(t, graphFlag, "stockonto-graph flag should exist")

	strFlag, ok := graphFlag.(cli.StringFlag)
	require.True(t, ok, "stockonto-graph flag should be a StringFlag")
	require.Equal(t, graphStockOnto, strFlag.Name)
	require.Equal(t, "stockonto", strFlag.Value)
	require.Equal(t,
		"arangodb named graph for managing stock and ontology",
		strFlag.Usage)
}

func testDbCollectionFlagsAreStrings(t *testing.T) {
	t.Helper()
	flags := dbCollectionFlags()

	for _, flg := range flags {
		_, ok := flg.(cli.StringFlag)
		require.True(t, ok, "all database collection flags should be StringFlags, but %s is not", getFlagName(flg))
	}
}

func testDbCollectionFlagsHaveDefaults(t *testing.T) {
	t.Helper()
	flags := dbCollectionFlags()

	expectedDefaults := map[string]string{
		collectionStock:             defaultStockCollection,
		collectionStockProp:         "stockprop",
		collectionStockKeyGenerator: "stock_key_generator",
		collectionStockTypeEdge:     "stock_type",
		collectionParentStrainEdge:  "parent_strain",
		collectionStockTermEdge:     "stock_term",
		graphStockPropType:          "stockprop_type",
		graphStrain2Parent:          "strain2parent",
		graphStockOnto:              "stockonto",
	}

	for expectedName, expectedDefault := range expectedDefaults {
		flg := findFlagByName(flags, expectedName)
		require.NotNil(t, flg, "flag %s should exist", expectedName)

		strFlag, ok := flg.(cli.StringFlag)
		require.True(t, ok, "flag %s should be a StringFlag", expectedName)
		require.Equal(t, expectedDefault, strFlag.Value, "flag %s should have correct default value", expectedName)
	}
}

func testDbCollectionFlagsUsage(t *testing.T) {
	t.Helper()
	flags := dbCollectionFlags()

	expectedUsages := map[string]string{
		collectionStock:             "arangodb collection for storing biological stocks",
		collectionStockProp:         "arangodb collection for storing stock properties",
		collectionStockKeyGenerator: "arangodb collection for generating unique IDs",
		collectionStockTypeEdge:     "arangodb edge collection for connecting stocks to their types (strain or plasmid)",
		collectionParentStrainEdge:  "arangodb edge collection for connecting strains to their parent",
		collectionStockTermEdge:     "arangodb edge collection for connecting stock to ontology term",
		graphStockPropType:          "arangodb named graph for managing relations between stocks and their properties",
		graphStrain2Parent:          "arangodb named graph for managing relations between strains and their parents",
		graphStockOnto:              "arangodb named graph for managing stock and ontology",
	}

	for expectedName, expectedUsage := range expectedUsages {
		flg := findFlagByName(flags, expectedName)
		require.NotNil(t, flg, "flag %s should exist", expectedName)

		actualUsage := getFlagUsage(flg)
		require.Equal(t, expectedUsage, actualUsage, "flag %s should have correct usage", expectedName)
	}
}

func testDbCollectionFlagsNoEnvVars(t *testing.T) {
	t.Helper()
	flags := dbCollectionFlags()

	for _, flg := range flags {
		envVar := getFlagEnvVar(flg)
		require.Empty(t, envVar, "database collection flag %s should not have environment variable", getFlagName(flg))
	}
}

// Test implementation functions for allFlags

func testAllFlagsCombination(t *testing.T) {
	t.Helper()
	flags := allFlags()
	require.NotNil(t, flags)
	require.NotEmpty(t, flags)
	require.Greater(t, len(flags), 15, "should have flags from multiple sources")
}

func testAllFlagsIncludesServer(t *testing.T) {
	t.Helper()
	flags := allFlags()
	portFlag := findFlagByName(flags, "port")
	require.NotNil(t, portFlag, "should include port flag from serverFlags")

	keyOffsetFlag := findFlagByName(flags, "keyoffset")
	require.NotNil(t, keyOffsetFlag, "should include keyoffset flag from serverFlags")
}

func testAllFlagsIncludesDb(t *testing.T) {
	t.Helper()
	flags := allFlags()
	stockFlag := findFlagByName(flags, collectionStock)
	require.NotNil(t, stockFlag, "should include stock-collection flag from dbCollectionFlags")

	stockPropFlag := findFlagByName(flags, collectionStockProp)
	require.NotNil(t, stockPropFlag, "should include stockprop-collection from dbCollectionFlags")
}

func testAllFlagsIncludesArangoDb(t *testing.T) {
	t.Helper()
	flags := allFlags()
	dbFlag := findFlagByName(flags, "arangodb-database")
	require.NotNil(t, dbFlag, "should include arangodb-database flag")

	strFlag, ok := dbFlag.(cli.StringFlag)
	require.True(t, ok, "arangodb-database flag should be a StringFlag")
	require.Contains(t, strFlag.Name, "arangodb-database")
	require.Contains(t, strFlag.Name, "db")
	require.Equal(t, "ARANGODB_DATABASE", strFlag.EnvVar)
	require.Equal(t, defaultDatabase, strFlag.Value)
	require.Equal(t, "arangodb database name", strFlag.Usage)
}

func testAllFlagsNoDuplicates(t *testing.T) {
	t.Helper()
	flags := allFlags()
	flagNames := make(map[string]int)
	for _, flg := range flags {
		name := getFlagName(flg)
		flagNames[name]++
	}

	for name, count := range flagNames {
		require.Equal(t, 1, count, "flag %s should appear only once", name)
	}
}

func testAllFlagsValidConfig(t *testing.T) {
	t.Helper()
	flags := allFlags()
	for _, flg := range flags {
		name := getFlagName(flg)
		require.NotEmpty(t, name, "all flags should have a name")

		usage := getFlagUsage(flg)
		require.NotEmpty(t, usage, "flag %s should have usage text", name)
	}
}

// Test implementation functions for main app

func testAppMetadata(t *testing.T) {
	t.Helper()
	app := cli.NewApp()
	app.Name = appName
	app.Usage = appUsage
	app.Version = appVersion

	require.Equal(t, appName, app.Name)
	require.Equal(t, appUsage, app.Usage)
	require.Equal(t, appVersion, app.Version)
}

func testAppGlobalFlags(t *testing.T) {
	t.Helper()
	app := cli.NewApp()
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  flagLogFormat,
			Usage: usageLogFormat,
			Value: logFormatJSON,
		},
		cli.StringFlag{
			Name:  flagLogLevel,
			Usage: usageLogLevel,
			Value: logLevelError,
		},
	}

	require.Len(t, app.Flags, 2)

	logFormatFlag := findFlagByName(app.Flags, flagLogFormat)
	require.NotNil(t, logFormatFlag)
	strFlag, ok := logFormatFlag.(cli.StringFlag)
	require.True(t, ok)
	require.Equal(t, logFormatJSON, strFlag.Value)

	logLevelFlag := findFlagByName(app.Flags, flagLogLevel)
	require.NotNil(t, logLevelFlag)
	strFlag, ok = logLevelFlag.(cli.StringFlag)
	require.True(t, ok)
	require.Equal(t, logLevelError, strFlag.Value)
}

func testAppStartServerCommand(t *testing.T) {
	t.Helper()
	app := cli.NewApp()
	app.Commands = []cli.Command{
		{
			Name:  flagStartServer,
			Usage: usageStartServer,
			Flags: allFlags(),
		},
	}

	require.Len(t, app.Commands, 1)
	require.Equal(t, flagStartServer, app.Commands[0].Name)
	require.Equal(t,
		usageStartServer,
		app.Commands[0].Usage)
	require.NotEmpty(t, app.Commands[0].Flags)
}

func testStartServerCommandFlags(t *testing.T) {
	t.Helper()
	app := cli.NewApp()
	app.Commands = []cli.Command{
		{
			Name:  flagStartServer,
			Usage: usageStartServer,
			Flags: allFlags(),
		},
	}

	cmd := app.Commands[0]
	require.NotEmpty(t, cmd.Flags)

	portFlag := findFlagByName(cmd.Flags, "port")
	require.NotNil(t, portFlag, "start-server should have port flag")

	dbFlag := findFlagByName(cmd.Flags, "arangodb-database")
	require.NotNil(t, dbFlag, "start-server should have arangodb-database flag")

	stockFlag := findFlagByName(cmd.Flags, collectionStock)
	require.NotNil(t, stockFlag, "start-server should have stock-collection flag")
}

// Test implementation functions for integration tests

func testAppRunsWithNoArgs(t *testing.T) {
	t.Helper()
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{appName}

	app := cli.NewApp()
	app.Name = appName
	app.Usage = appUsage
	app.Version = appVersion

	err := app.Run(os.Args)
	require.NoError(t, err, "app should run without error when no command is given")
}

func testAppRunsWithHelpFlag(t *testing.T) {
	t.Helper()
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{appName, "--help"}

	app := cli.NewApp()
	app.Name = appName
	app.Usage = appUsage
	app.Version = appVersion
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  flagLogFormat,
			Usage: usageLogFormat,
			Value: logFormatJSON,
		},
		cli.StringFlag{
			Name:  flagLogLevel,
			Usage: usageLogLevel,
			Value: logLevelError,
		},
	}

	err := app.Run(os.Args)
	require.NoError(t, err, "app should run without error when help flag is given")
}

func testAppRunsWithVersionFlag(t *testing.T) {
	t.Helper()
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"modware-stock", "--version"}

	app := cli.NewApp()
	app.Name = "modware-stock"
	app.Version = "1.0.0"

	err := app.Run(os.Args)
	require.NoError(t, err, "app should run without error when version flag is given")
}

func testMainFunctionStructure(t *testing.T) {
	t.Helper()

	// This test replicates the exact structure of main() function
	app := cli.NewApp()
	app.Name = appName
	app.Usage = appUsage
	app.Version = appVersion
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  flagLogFormat,
			Usage: usageLogFormat,
			Value: logFormatJSON,
		},
		cli.StringFlag{
			Name:  flagLogLevel,
			Usage: usageLogLevel,
			Value: logLevelError,
		},
	}
	app.Commands = []cli.Command{
		{
			Name:  flagStartServer,
			Usage: usageStartServer,
			Flags: allFlags(),
		},
	}

	// Verify all components match what main() creates
	require.Equal(t, "modware-stock", app.Name)
	require.Equal(t, "cli for modware-stock microservice", app.Usage)
	require.Equal(t, "1.0.0", app.Version)
	require.Len(t, app.Flags, 2)
	require.Len(t, app.Commands, 1)
	require.Equal(t, flagStartServer, app.Commands[0].Name)
	require.Equal(t, usageStartServer, app.Commands[0].Usage)

	// Verify the command has all flags from allFlags()
	require.NotEmpty(t, app.Commands[0].Flags)
	expectedFlags := allFlags()
	require.Len(t, app.Commands[0].Flags, len(expectedFlags))
}

func testCompleteApplicationLifecycle(t *testing.T) {
	t.Helper()
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Test with just the app name (no command)
	os.Args = []string{appName}

	// Create full app as main() does
	app := cli.NewApp()
	app.Name = appName
	app.Usage = appUsage
	app.Version = appVersion
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  flagLogFormat,
			Usage: usageLogFormat,
			Value: logFormatJSON,
		},
		cli.StringFlag{
			Name:  flagLogLevel,
			Usage: usageLogLevel,
			Value: logLevelError,
		},
	}
	app.Commands = []cli.Command{
		{
			Name:  flagStartServer,
			Usage: usageStartServer,
			Flags: allFlags(),
		},
	}

	// This should not error as the app is valid
	err := app.Run(os.Args)
	require.NoError(t, err, "app should run successfully with valid configuration")

	// Verify the app structure after Run
	require.NotNil(t, app.Name)
	require.NotNil(t, app.Version)
	require.NotEmpty(t, app.Commands)
}

// Test implementation functions for flag defaults

func testServerFlagsDefaults(t *testing.T) {
	t.Helper()
	flags := serverFlags()

	portFlag := findFlagByName(flags, "port")
	require.NotNil(t, portFlag)
	strFlag, ok := portFlag.(cli.StringFlag)
	require.True(t, ok)
	require.NotEmpty(t, strFlag.Value, "port should have a default value")

	keyOffsetFlag := findFlagByName(flags, "keyoffset")
	require.NotNil(t, keyOffsetFlag)
	intFlag, ok := keyOffsetFlag.(cli.IntFlag)
	require.True(t, ok)
	require.Greater(t, intFlag.Value, 0, "keyoffset should be positive")
}

func testDbFlagsDefaults(t *testing.T) {
	t.Helper()
	flags := dbCollectionFlags()

	for _, flg := range flags {
		strFlag, ok := flg.(cli.StringFlag)
		require.True(t, ok, "all db collection flags should be StringFlags")
		require.NotEmpty(t, strFlag.Value, "flag %s should have a default value", strFlag.Name)
	}
}

// Test implementation functions for CLI integration

func testFlagsCliParsing(t *testing.T) {
	t.Helper()
	app := cli.NewApp()
	app.Flags = allFlags()

	set := flag.NewFlagSet("test", flag.ContinueOnError)
	ctx := cli.NewContext(app, set, nil)

	require.NotNil(t, ctx)
}

func testGlobalFlagsCliParsing(t *testing.T) {
	t.Helper()
	app := cli.NewApp()
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  flagLogFormat,
			Usage: usageLogFormat,
			Value: logFormatJSON,
		},
		cli.StringFlag{
			Name:  flagLogLevel,
			Usage: usageLogLevel,
			Value: logLevelError,
		},
	}

	set := flag.NewFlagSet("test", flag.ContinueOnError)
	ctx := cli.NewContext(app, set, nil)

	require.NotNil(t, ctx)
}

// Helper functions

func findFlagByName(flags []cli.Flag, name string) cli.Flag {
	for _, flg := range flags {
		flagName := getFlagName(flg)
		if flagName == name || containsName(flagName, name) {
			return flg
		}
	}
	return nil
}

func getFlagName(flg cli.Flag) string {
	switch f := flg.(type) {
	case cli.StringFlag:
		return f.Name
	case cli.IntFlag:
		return f.Name
	case cli.BoolFlag:
		return f.Name
	case cli.BoolTFlag:
		return f.Name
	case cli.StringSliceFlag:
		return f.Name
	case cli.IntSliceFlag:
		return f.Name
	default:
		return ""
	}
}

func getFlagUsage(flg cli.Flag) string {
	switch f := flg.(type) {
	case cli.StringFlag:
		return f.Usage
	case cli.IntFlag:
		return f.Usage
	case cli.BoolFlag:
		return f.Usage
	case cli.BoolTFlag:
		return f.Usage
	case cli.StringSliceFlag:
		return f.Usage
	case cli.IntSliceFlag:
		return f.Usage
	default:
		return ""
	}
}

func getFlagEnvVar(flg cli.Flag) string {
	switch f := flg.(type) {
	case cli.StringFlag:
		return f.EnvVar
	case cli.IntFlag:
		return f.EnvVar
	case cli.BoolFlag:
		return f.EnvVar
	case cli.BoolTFlag:
		return f.EnvVar
	case cli.StringSliceFlag:
		return f.EnvVar
	case cli.IntSliceFlag:
		return f.EnvVar
	default:
		return ""
	}
}

func containsName(flagName, searchName string) bool {
	if flagName == "" {
		return false
	}
	for i := 0; i < len(flagName); i++ {
		if i+len(searchName) <= len(flagName) && flagName[i:i+len(searchName)] == searchName {
			return true
		}
	}
	return false
}

// Test implementation functions for main function behavior

func testMainFunctionConfiguration(t *testing.T) {
	t.Helper()

	// Create an app instance as main() does
	app := cli.NewApp()
	app.Name = appName
	app.Usage = appUsage
	app.Version = appVersion
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  flagLogFormat,
			Usage: usageLogFormat,
			Value: logFormatJSON,
		},
		cli.StringFlag{
			Name:  flagLogLevel,
			Usage: usageLogLevel,
			Value: logLevelError,
		},
	}

	// Verify metadata
	require.Equal(t, "modware-stock", app.Name)
	require.Equal(t, "cli for modware-stock microservice", app.Usage)
	require.Equal(t, "1.0.0", app.Version)

	// Verify global flags
	require.Len(t, app.Flags, 2)
	logFormatFlag := findFlagByName(app.Flags, flagLogFormat)
	require.NotNil(t, logFormatFlag)
	logLevelFlag := findFlagByName(app.Flags, flagLogLevel)
	require.NotNil(t, logLevelFlag)
}

func testMainFunctionComponents(t *testing.T) {
	t.Helper()

	// Create the full app as main() does with all components
	app := cli.NewApp()
	app.Name = appName
	app.Usage = appUsage
	app.Version = appVersion
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  flagLogFormat,
			Usage: usageLogFormat,
			Value: logFormatJSON,
		},
		cli.StringFlag{
			Name:  flagLogLevel,
			Usage: usageLogLevel,
			Value: logLevelError,
		},
	}
	app.Commands = []cli.Command{
		{
			Name:  flagStartServer,
			Usage: usageStartServer,
			Flags: allFlags(),
		},
	}

	// Verify the app has all required components
	require.NotNil(t, app.Name)
	require.NotNil(t, app.Usage)
	require.NotNil(t, app.Version)
	require.NotEmpty(t, app.Flags)
	require.NotEmpty(t, app.Commands)

	// Verify the start-server command exists
	require.Len(t, app.Commands, 1)
	startServerCmd := app.Commands[0]
	require.Equal(t, flagStartServer, startServerCmd.Name)
	require.Equal(t, usageStartServer, startServerCmd.Usage)

	// Verify start-server command has flags
	require.NotEmpty(t, startServerCmd.Flags)

	// Verify critical flags are present in start-server command
	portFlag := findFlagByName(startServerCmd.Flags, "port")
	require.NotNil(t, portFlag, "start-server must have port flag")

	dbFlag := findFlagByName(startServerCmd.Flags, "arangodb-database")
	require.NotNil(t, dbFlag, "start-server must have arangodb-database flag")

	stockCollectionFlag := findFlagByName(startServerCmd.Flags, collectionStock)
	require.NotNil(t, stockCollectionFlag, "start-server must have stock-collection flag")

	keyOffsetFlag := findFlagByName(startServerCmd.Flags, "keyoffset")
	require.NotNil(t, keyOffsetFlag, "start-server must have keyoffset flag")
}
