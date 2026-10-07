package service

// Stock type names used when building stock API records.
const (
	stockTypePlasmid = "plasmid"
	stockTypeStrain  = "strain"
	// autocompleteStockType is the resource type of an autocomplete
	// request.
	autocompleteStockType = "stock"
	// autocompleteFieldStockID is the repository field label that the
	// stock identifier suggestion carries.
	autocompleteFieldStockID = "stock_id"
	// autocompleteFieldLabel is the repository field label that the
	// label suggestion carries.
	autocompleteFieldLabel = "label"
	// autocompleteFieldName is the repository field label that the
	// plasmid name suggestion carries.
	autocompleteFieldName = "name"
)
