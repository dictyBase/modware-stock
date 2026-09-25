package arangodb

// FMap maps filters to database fields
var FMap = map[string]string{
	fieldCreatedAt: "s.created_at",
	fieldUpdatedAt: "s.updated_at",
	fieldDepositor: "s.depositor",
	fieldSummary:   "s.summary",
	"id":           "s.stock_id",
	"gene":         "s.genes",
	fieldPlasmid:   "stock_prop.plasmid",
	fieldSpecies:   "stock_prop.species",
	paramName:      "stock_prop.names",
	fieldLabel:     "stock_prop.label",
	paramOntology:  "cv.metadata.namespace",
	"tag":          "cvterm.label",
	paramParent:    paramParent,
	"plasmid_name": "stock_prop.name",
}
