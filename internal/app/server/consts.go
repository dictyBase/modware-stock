package server

// CLI flag names read by the server.
const (
	flagLogFormat    = "log-format"
	flagLogLevel     = "log-level"
	flagArangodbPort = "arangodb-port"
	flagIsSecure     = "is-secure"
)

// Accepted logging option values.
const (
	logFormatText = "text"
	logFormatJSON = "json"
	logLevelError = "error"
)
