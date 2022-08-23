package arangodb

// FMap maps filters to database fields
var FMap = map[string]string{
	"created_at":   "s.created_at",
	"updated_at":   "s.updated_at",
	"depositor":    "s.depositor",
	"summary":      "s.summary",
	"id":           "s.stock_id",
	"gene":         "s.genes",
	"plasmid":      "stock_prop.plasmid",
	"species":      "stock_prop.species",
	"name":         "stock_prop.names",
	"label":        "stock_prop.label",
	"ontology":     "cv.metadata.namespace",
	"tag":          "cvterm.label",
	"parent":       "parent",
	"plasmid_name": "name",
}
