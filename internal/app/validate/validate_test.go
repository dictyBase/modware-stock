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

func getValidateServerArgsTestCases() []validateServerArgsTestCase {
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

func getEmptyStringTestCases() []validateServerArgsTestCase {
	return []validateServerArgsTestCase{
		{
			name: "empty string for arangodb-pass",
			flagValues: map[string]string{
				"arangodb-pass":     "",
				"arangodb-database": "testdb",
				"arangodb-user":     "testuser",
				"nats-host":         "localhost",
				"nats-port":         "4222",
			},
			expectError: true,
			errorMsg:    "argument arangodb-pass is missing",
		},
		{
			name: "empty string for arangodb-database",
			flagValues: map[string]string{
				"arangodb-pass":     "password123",
				"arangodb-database": "",
				"arangodb-user":     "testuser",
				"nats-host":         "localhost",
				"nats-port":         "4222",
			},
			expectError: true,
			errorMsg:    "argument arangodb-database is missing",
		},
		{
			name: "empty string for arangodb-user",
			flagValues: map[string]string{
				"arangodb-pass":     "password123",
				"arangodb-database": "testdb",
				"arangodb-user":     "",
				"nats-host":         "localhost",
				"nats-port":         "4222",
			},
			expectError: true,
			errorMsg:    "argument arangodb-user is missing",
		},
		{
			name: "empty string for nats-host",
			flagValues: map[string]string{
				"arangodb-pass":     "password123",
				"arangodb-database": "testdb",
				"arangodb-user":     "testuser",
				"nats-host":         "",
				"nats-port":         "4222",
			},
			expectError: true,
			errorMsg:    "argument nats-host is missing",
		},
		{
			name: "empty string for nats-port",
			flagValues: map[string]string{
				"arangodb-pass":     "password123",
				"arangodb-database": "testdb",
				"arangodb-user":     "testuser",
				"nats-host":         "localhost",
				"nats-port":         "",
			},
			expectError: true,
			errorMsg:    "argument nats-port is missing",
		},
	}
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

func TestValidateServerArgs(t *testing.T) {
	t.Parallel()

	tests := getValidateServerArgsTestCases()

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			flagSet := createFlagSetWithValues(testCase.flagValues)
			ctx := cli.NewContext(nil, flagSet, nil)
			err := ValidateServerArgs(ctx)

			if testCase.expectError {
				assertValidationError(t, err, testCase.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateServerArgs_AllParametersOrder(t *testing.T) {
	t.Parallel()

	// Test that validation checks parameters in the documented order
	// by providing only later parameters and expecting first missing one
	testCases := []struct {
		name          string
		missingParam  string
		providedFlags []string
	}{
		{
			name:          "first parameter missing",
			missingParam:  "arangodb-pass",
			providedFlags: []string{"arangodb-database", "arangodb-user", "nats-host", "nats-port"},
		},
		{
			name:          "second parameter missing",
			missingParam:  "arangodb-database",
			providedFlags: []string{"arangodb-pass", "arangodb-user", "nats-host", "nats-port"},
		},
		{
			name:          "third parameter missing",
			missingParam:  "arangodb-user",
			providedFlags: []string{"arangodb-pass", "arangodb-database", "nats-host", "nats-port"},
		},
		{
			name:          "fourth parameter missing",
			missingParam:  "nats-host",
			providedFlags: []string{"arangodb-pass", "arangodb-database", "arangodb-user", "nats-port"},
		},
		{
			name:          "fifth parameter missing",
			missingParam:  "nats-port",
			providedFlags: []string{"arangodb-pass", "arangodb-database", "arangodb-user", "nats-host"},
		},
	}

	allFlags := []string{"arangodb-pass", "arangodb-database", "arangodb-user", "nats-host", "nats-port"}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			flagSet := flag.NewFlagSet("test", flag.ContinueOnError)

			// Register all flags
			for _, flagName := range allFlags {
				flagSet.String(flagName, "", "")
			}

			// Set only the provided flags
			for _, flagName := range testCase.providedFlags {
				err := flagSet.Set(flagName, "value")
				require.NoError(t, err)
			}

			ctx := cli.NewContext(nil, flagSet, nil)
			err := ValidateServerArgs(ctx)

			require.Error(t, err)
			exitErr, ok := err.(*cli.ExitError)
			require.True(t, ok)
			require.Equal(t, 2, exitErr.ExitCode())
			require.Contains(t, exitErr.Error(), testCase.missingParam)
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

func testValidateServerArgsWithValues(t *testing.T, values map[string]string) {
	t.Helper()
	flagSet := createFlagSetWithValues(values)
	ctx := cli.NewContext(nil, flagSet, nil)
	err := ValidateServerArgs(ctx)
	require.NoError(t, err)
}

func TestValidateServerArgs_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("very long parameter values", func(t *testing.T) {
		t.Parallel()

		longValue := createLongValue(10000)
		testValidateServerArgsWithValues(t, map[string]string{
			"arangodb-pass":     longValue,
			"arangodb-database": longValue,
			"arangodb-user":     longValue,
			"nats-host":         longValue,
			"nats-port":         longValue,
		})
	})

	t.Run("special characters in values", func(t *testing.T) {
		t.Parallel()

		testValidateServerArgsWithValues(t, map[string]string{
			"arangodb-pass":     "p@$$w0rd!#%&*(){}[]|\\:;\"'<>,.?/~`",
			"arangodb-database": "db-name_123",
			"arangodb-user":     "user@domain.com",
			"nats-host":         "nats://localhost:4222",
			"nats-port":         "4222",
		})
	})

	t.Run("unicode characters in values", func(t *testing.T) {
		t.Parallel()

		testValidateServerArgsWithValues(t, map[string]string{
			"arangodb-pass":     "密码123",
			"arangodb-database": "データベース",
			"arangodb-user":     "usuario",
			"nats-host":         "localhost",
			"nats-port":         "4222",
		})
	})
}
