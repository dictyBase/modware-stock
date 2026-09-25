package arangodb

// Bind parameter keys used across AQL queries.
const (
	bindCVCollection        = "@cv_collection"
	bindCvtermCollection    = "@cvterm_collection"
	bindStockCollection     = "@stock_collection"
	bindStockTypeCollection = "@stock_type_collection"
	bindStockPropCollection = "@stock_properties_collection"
	bindStockTermCollection = "@stock_term_collection"
	bindStockKeyGenerator   = "@stock_key_generator"
)

// Collection and graph names referenced by queries and map keys.
const (
	nameStockCollection   = "stock_collection"
	nameStockPropGraph    = "stock_prop_graph"
	nameStockCvtermGraph  = "stock_cvterm_graph"
	nameParentGraph       = "parent_graph"
	nameStockKeyGenerator = "stock_key_generator_collection"
)

// Query bind parameter names.
const (
	paramStockID  = "stock_id"
	paramOntology = "ontology"
	paramLimit    = "limit"
	paramKey      = "key"
	paramName     = "name"
	paramParent   = "parent"
)

// Attribute/field names used in bind params, filter maps and property maps.
const (
	fieldCreatedAt       = "created_at"
	fieldUpdatedAt       = "updated_at"
	fieldCreatedBy       = "created_by"
	fieldUpdatedBy       = "updated_by"
	fieldDepositor       = "depositor"
	fieldSummary         = "summary"
	fieldEditableSummary = "editable_summary"
	fieldSpecies         = "species"
	fieldLabel           = "label"
	fieldGenes           = "genes"
	fieldDbxrefs         = "dbxrefs"
	fieldPublications    = "publications"
	fieldPlasmid         = "plasmid"
)
