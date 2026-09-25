package main

// CLI flag names shared by the application and its tests.
const (
	flagLogFormat   = "log-format"
	flagLogLevel    = "log-level"
	flagStartServer = "start-server"
)

// CLI flag usage strings.
const (
	usageLogFormat   = "format of the logging out, either of json or text."
	usageLogLevel    = "log level for the application"
	usageStartServer = "starts the modware-stock microservice with grpc backends"
)

// Accepted values for the logging flags.
const (
	logFormatJSON = "json"
	logLevelError = "error"
)

// Default arangodb database and stock collection names.
const (
	defaultDatabase        = "stock"
	defaultStockCollection = "stock"
)

// Arangodb collection and graph names.
const (
	collectionStock             = "stock-collection"
	collectionStockProp         = "stockprop-collection"
	collectionStockKeyGenerator = "stock-key-generator-collection"
	collectionStockTypeEdge     = "stock-type-edge"
	collectionParentStrainEdge  = "parent-strain-edge"
	collectionStockTermEdge     = "stock-term-edge"
	graphStockPropType          = "stockproptype-graph"
	graphStrain2Parent          = "strain2parent-graph"
	graphStockOnto              = "stockonto-graph"
)
