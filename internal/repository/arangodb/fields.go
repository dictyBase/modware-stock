package arangodb

// FMap maps filters to database fields
var FMap = map[string]string{
	"created_at":   "created_at",
	"updated_at":   "updated_at",
	"depositor":    "depositor",
	"summary":      "summary",
	"plasmid":      "plasmid",
	"species":      "species",
	"name":         "names",
	"parent":       "parent",
	"plasmid_name": "name",
	"id":           "stock_id",
	"label":        "label",
}
