package arangodb

// mapping of filters to database fields
var fmap = map[string]string{
	"created_at":      "created_at",
	"updated_at":      "updated_at",
	"depositor":       "depositor",
	"summary":         "summary",
	"plasmid":         "plasmid",
	"species":         "species",
	"systematic_name": "systematic_name",
	"name":            "names",
	"stock_type":      "type",
	// "parent":          "parents",
}