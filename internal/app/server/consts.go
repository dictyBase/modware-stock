package server

const (
	// CLI flag names read by the server.
	flagLogFormat    = "log-format"
	flagLogLevel     = "log-level"
	flagArangodbPort = "arangodb-port"
	flagIsSecure     = "is-secure"

	// Accepted logging option values.
	logFormatText = "text"
	logFormatJSON = "json"
	logLevelError = "error"
)
