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
				"arangodb-pass":     "password123",
				"arangodb-database": "testdb",
				"arangodb-user":     "testuser",
				"nats-host":         "localhost",
				"nats-port":         "4222",
			},
			expectError: false,
		},
		{
			name:        "all parameters missing - fails on first",
			flagValues:  map[string]string{},
			expectError: true,
			errorMsg:    "argument arangodb-pass is missing",
		},
		{
			name: "whitespace values treated as present",
			flagValues: map[string]string{
				"arangodb-pass":     " ",
				"arangodb-database": "testdb",
				"arangodb-user":     "testuser",
				"nats-host":         "localhost",
				"nats-port":         "4222",
			},
			expectError: false,
		},
		{
			name: "multiple missing parameters - first one in sequence",
			flagValues: map[string]string{
				"arangodb-user": "testuser",
				"nats-port":     "4222",
			},
			expectError: true,
			errorMsg:    "argument arangodb-pass is missing",
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
				"arangodb-database": "testdb",
				"arangodb-user":     "testuser",
				"nats-host":         "localhost",
				"nats-port":         "4222",
			},
			expectError: true,
			errorMsg:    "argument arangodb-pass is missing",
		},
		{
			name: "missing arangodb-database",
			flagValues: map[string]string{
				"arangodb-pass": "password123",
				"arangodb-user": "testuser",
				"nats-host":     "localhost",
				"nats-port":     "4222",
			},
			expectError: true,
			errorMsg:    "argument arangodb-database is missing",
		},
		{
			name: "missing arangodb-user",
			flagValues: map[string]string{
				"arangodb-pass":     "password123",
				"arangodb-database": "testdb",
				"nats-host":         "localhost",
				"nats-port":         "4222",
			},
			expectError: true,
			errorMsg:    "argument arangodb-user is missing",
		},
		{
			name: "missing nats-host",
			flagValues: map[string]string{
				"arangodb-pass":     "password123",
				"arangodb-database": "testdb",
				"arangodb-user":     "testuser",
				"nats-port":         "4222",
			},
			expectError: true,
			errorMsg:    "argument nats-host is missing",
		},
		{
			name: "missing nats-port",
			flagValues: map[string]string{
				"arangodb-pass":     "password123",
				"arangodb-database": "testdb",
				"arangodb-user":     "testuser",
				"nats-host":         "localhost",
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
		"arangodb-pass":     "password123",
		"arangodb-database": "testdb",
		"arangodb-user":     "testuser",
		"nats-host":         "localhost",
		"nats-port":         "4222",
	}

	paramNames := []string{"arangodb-pass", "arangodb-database", "arangodb-user", "nats-host", "nats-port"}
	cases := make([]validateServerArgsTestCase, 0, len(paramNames))
	for _, param := range paramNames {
		cases = append(cases, buildEmptyStringTestCase(param, allParams))
	}
	return cases
}

func createFlagSetWithValues(flagValues map[string]string) *flag.FlagSet {
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	flagSet.String("arangodb-pass", "", "")
	flagSet.String("arangodb-database", "", "")
	flagSet.String("arangodb-user", "", "")
	flagSet.String("nats-host", "", "")
	flagSet.String("nats-port", "", "")

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

	allFlags := []string{"arangodb-pass", "arangodb-database", "arangodb-user", "nats-host", "nats-port"}

	// Test that validation checks parameters in the documented order
	testCases := []struct {
		name          string
		missingParam  string
		providedFlags []string
	}{
		{"first parameter missing", "arangodb-pass", allFlags[1:]},
		{"second parameter missing", "arangodb-database", []string{allFlags[0], allFlags[2], allFlags[3], allFlags[4]}},
		{"third parameter missing", "arangodb-user", []string{allFlags[0], allFlags[1], allFlags[3], allFlags[4]}},
		{"fourth parameter missing", "nats-host", allFlags[:3]},
		{"fifth parameter missing", "nats-port", allFlags[:4]},
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
			"arangodb-pass":     longValue,
			"arangodb-database": longValue,
			"arangodb-user":     longValue,
			"nats-host":         longValue,
			"nats-port":         longValue,
		})
	})

	t.Run("special characters in values", func(t *testing.T) {
		t.Parallel()

		testServerArgsWithValues(t, map[string]string{
			"arangodb-pass":     "p@$$w0rd!#%&*(){}[]|\\:;\"'<>,.?/~`",
			"arangodb-database": "db-name_123",
			"arangodb-user":     "user@domain.com",
			"nats-host":         "nats://localhost:4222",
			"nats-port":         "4222",
		})
	})

	t.Run("unicode characters in values", func(t *testing.T) {
		t.Parallel()

		testServerArgsWithValues(t, map[string]string{
			"arangodb-pass":     "密码123",
			"arangodb-database": "データベース",
			"arangodb-user":     "usuario",
			"nats-host":         "localhost",
			"nats-port":         "4222",
		})
	})
}
