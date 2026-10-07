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
	// fullSearchFieldSummary is the repository field label of the
	// summary prose field of the full search.
	fullSearchFieldSummary = "summary"
	// fullSearchFieldDepositor is the repository field label of the
	// depositor prose field of the full search.
	fullSearchFieldDepositor = "depositor"
	// searchFieldGenes is the repository field label of the genes
	// array field.
	searchFieldGenes = "genes"
	// searchFieldDbxrefs is the repository field label of the dbxrefs
	// array field.
	searchFieldDbxrefs = "dbxrefs"
	// searchFieldNames is the repository field label of the names
	// array field.
	searchFieldNames = "names"
	// searchFieldSpecies is the repository field label of the species
	// scalar field.
	searchFieldSpecies = "species"
	// searchFieldPlasmid is the repository field label of the plasmid
	// scalar field.
	searchFieldPlasmid = "plasmid"
)
