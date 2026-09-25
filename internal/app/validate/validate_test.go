package validate

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli"
)

type validateServerArgsTestCase struct {
	name        string
	flagValues  map[string]string
	expectError bool
	errorMsg    string
}

func getServerArgsTestCases() []validateServerArgsTestCase {
	cases := []validateServerArgsTestCase{
		{
			name: "all required parameters present",
			flagValues: map[string]string{
				flagArangodbPass:     testPassword,
				flagArangodbDatabase: testDatabase,
				flagArangodbUser:     testUser,
				flagNatsHost:         testNatsHost,
				flagNatsPort:         testNatsPort,
			},
			expectError: false,
		},
		{
			name:        "all parameters missing - fails on first",
			flagValues:  map[string]string{},
			expectError: true,
			errorMsg:    testMissingArangodbPassMsg,
		},
		{
			name: "whitespace values treated as present",
			flagValues: map[string]string{
				flagArangodbPass:     " ",
				flagArangodbDatabase: testDatabase,
				flagArangodbUser:     testUser,
				flagNatsHost:         testNatsHost,
				flagNatsPort:         testNatsPort,
			},
			expectError: false,
		},
		{
			name: "multiple missing parameters - first one in sequence",
			flagValues: map[string]string{
				flagArangodbUser: testUser,
				flagNatsPort:     testNatsPort,
			},
			expectError: true,
			errorMsg:    testMissingArangodbPassMsg,
		},
	}

	cases = append(cases, getMissingParameterTestCases()...)
	cases = append(cases, getEmptyStringTestCases()...)

	return cases
}

func getMissingParameterTestCases() []validateServerArgsTestCase {
	return []validateServerArgsTestCase{
		{
			name: "missing arangodb-pass",
			flagValues: map[string]string{
				flagArangodbDatabase: testDatabase,
				flagArangodbUser:     testUser,
				flagNatsHost:         testNatsHost,
				flagNatsPort:         testNatsPort,
			},
			expectError: true,
			errorMsg:    testMissingArangodbPassMsg,
		},
		{
			name: "missing arangodb-database",
			flagValues: map[string]string{
				flagArangodbPass: testPassword,
				flagArangodbUser: testUser,
				flagNatsHost:     testNatsHost,
				flagNatsPort:     testNatsPort,
			},
			expectError: true,
			errorMsg:    "argument arangodb-database is missing",
		},
		{
			name: "missing arangodb-user",
			flagValues: map[string]string{
				flagArangodbPass:     testPassword,
				flagArangodbDatabase: testDatabase,
				flagNatsHost:         testNatsHost,
				flagNatsPort:         testNatsPort,
			},
			expectError: true,
			errorMsg:    "argument arangodb-user is missing",
		},
		{
			name: "missing nats-host",
			flagValues: map[string]string{
				flagArangodbPass:     testPassword,
				flagArangodbDatabase: testDatabase,
				flagArangodbUser:     testUser,
				flagNatsPort:         testNatsPort,
			},
			expectError: true,
			errorMsg:    "argument nats-host is missing",
		},
		{
			name: "missing nats-port",
			flagValues: map[string]string{
				flagArangodbPass:     testPassword,
				flagArangodbDatabase: testDatabase,
				flagArangodbUser:     testUser,
				flagNatsHost:         testNatsHost,
			},
			expectError: true,
			errorMsg:    "argument nats-port is missing",
		},
	}
}

func buildEmptyStringTestCase(paramName string, allParams map[string]string) validateServerArgsTestCase {
	flagValues := make(map[string]string)
	for k, v := range allParams {
		if k == paramName {
			flagValues[k] = ""
		} else {
			flagValues[k] = v
		}
	}
	return validateServerArgsTestCase{
		name:        "empty string for " + paramName,
		flagValues:  flagValues,
		expectError: true,
		errorMsg:    "argument " + paramName + " is missing",
	}
}

func getEmptyStringTestCases() []validateServerArgsTestCase {
	allParams := map[string]string{
		flagArangodbPass:     testPassword,
		flagArangodbDatabase: testDatabase,
		flagArangodbUser:     testUser,
		flagNatsHost:         testNatsHost,
		flagNatsPort:         testNatsPort,
	}

	paramNames := []string{flagArangodbPass, flagArangodbDatabase, flagArangodbUser, flagNatsHost, flagNatsPort}
	cases := make([]validateServerArgsTestCase, 0, len(paramNames))
	for _, param := range paramNames {
		cases = append(cases, buildEmptyStringTestCase(param, allParams))
	}
	return cases
}

func createFlagSetWithValues(flagValues map[string]string) *flag.FlagSet {
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	flagSet.String(flagArangodbPass, "", "")
	flagSet.String(flagArangodbDatabase, "", "")
	flagSet.String(flagArangodbUser, "", "")
	flagSet.String(flagNatsHost, "", "")
	flagSet.String(flagNatsPort, "", "")

	for key, value := range flagValues {
		_ = flagSet.Set(key, value)
	}

	return flagSet
}

func assertValidationError(t *testing.T, err error, expectedMsg string) {
	t.Helper()
	require.Error(t, err)
	exitErr, ok := err.(*cli.ExitError)
	require.True(t, ok, "Expected cli.ExitError, got %T", err)
	require.Equal(t, 2, exitErr.ExitCode())
	require.Contains(t, exitErr.Error(), expectedMsg)
}

func TestServerArgs(t *testing.T) {
	t.Parallel()

	tests := getServerArgsTestCases()

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			flagSet := createFlagSetWithValues(testCase.flagValues)
			ctx := cli.NewContext(nil, flagSet, nil)
			err := ServerArgs(ctx)

			if testCase.expectError {
				assertValidationError(t, err, testCase.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func createFlagSetForParameterOrder(allFlags, providedFlags []string) *flag.FlagSet {
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	for _, flagName := range allFlags {
		flagSet.String(flagName, "", "")
	}
	for _, flagName := range providedFlags {
		_ = flagSet.Set(flagName, "value")
	}
	return flagSet
}

func assertParameterOrderError(t *testing.T, err error, missingParam string) {
	t.Helper()
	require.Error(t, err)
	exitErr, ok := err.(*cli.ExitError)
	require.True(t, ok)
	require.Equal(t, 2, exitErr.ExitCode())
	require.Contains(t, exitErr.Error(), missingParam)
}

func TestServerArgs_AllParametersOrder(t *testing.T) {
	t.Parallel()

	allFlags := []string{flagArangodbPass, flagArangodbDatabase, flagArangodbUser, flagNatsHost, flagNatsPort}

	// Test that validation checks parameters in the documented order
	testCases := []struct {
		name          string
		missingParam  string
		providedFlags []string
	}{
		{"first parameter missing", flagArangodbPass, allFlags[1:]},
		{"second parameter missing", flagArangodbDatabase, []string{allFlags[0], allFlags[2], allFlags[3], allFlags[4]}},
		{"third parameter missing", flagArangodbUser, []string{allFlags[0], allFlags[1], allFlags[3], allFlags[4]}},
		{"fourth parameter missing", flagNatsHost, allFlags[:3]},
		{"fifth parameter missing", flagNatsPort, allFlags[:4]},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			flagSet := createFlagSetForParameterOrder(allFlags, testCase.providedFlags)
			ctx := cli.NewContext(nil, flagSet, nil)
			err := ServerArgs(ctx)
			assertParameterOrderError(t, err, testCase.missingParam)
		})
	}
}

func createLongValue(size int) string {
	longValue := make([]byte, size)
	for i := range longValue {
		longValue[i] = 'a'
	}
	return string(longValue)
}

func testServerArgsWithValues(t *testing.T, values map[string]string) {
	t.Helper()
	flagSet := createFlagSetWithValues(values)
	ctx := cli.NewContext(nil, flagSet, nil)
	err := ServerArgs(ctx)
	require.NoError(t, err)
}

func TestServerArgs_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("very long parameter values", func(t *testing.T) {
		t.Parallel()

		longValue := createLongValue(10000)
		testServerArgsWithValues(t, map[string]string{
			flagArangodbPass:     longValue,
			flagArangodbDatabase: longValue,
			flagArangodbUser:     longValue,
			flagNatsHost:         longValue,
			flagNatsPort:         longValue,
		})
	})

	t.Run("special characters in values", func(t *testing.T) {
		t.Parallel()

		testServerArgsWithValues(t, map[string]string{
			flagArangodbPass:     "p@$$w0rd!#%&*(){}[]|\\:;\"'<>,.?/~`",
			flagArangodbDatabase: "db-name_123",
			flagArangodbUser:     "user@domain.com",
			flagNatsHost:         "nats://localhost:4222",
			flagNatsPort:         testNatsPort,
		})
	})

	t.Run("unicode characters in values", func(t *testing.T) {
		t.Parallel()

		testServerArgsWithValues(t, map[string]string{
			flagArangodbPass:     "密码123",
			flagArangodbDatabase: "データベース",
			flagArangodbUser:     "usuario",
			flagNatsHost:         testNatsHost,
			flagNatsPort:         testNatsPort,
		})
	})
}
