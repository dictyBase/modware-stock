package server

import "time"

const (
	// CLI flag names read by the server.
	flagLogFormat    = "log-format"
	flagLogLevel     = "log-level"
	flagArangodbPort = "arangodb-port"
	flagIsSecure     = "is-secure"
	flagRedisHost    = "redis-master-service-host"
	flagRedisPort    = "redis-master-service-port"

	// Accepted logging option values.
	logFormatText = "text"
	logFormatJSON = "json"
	logLevelError = "error"

	// Multiplexed server constants.
	contentTypeHeader       = "Content-Type"
	grpcContentTypePrefix   = "application/grpc"
	httpProtoMajor2         = 2
	serverReadHeaderTimeout = 10 * time.Second
)
