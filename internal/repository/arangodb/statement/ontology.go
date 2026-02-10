package statement

// AQL queries for ontology term operations
const (
	// StrainExistTermQ checks if an ontology term exists for strain characteristics
	StrainExistTermQ = `
		FOR cv IN @@cv_collection
			FOR cvt IN @@cvterm_collection
				FILTER cv.metadata.namespace == @ontology
				FILTER cvt.label == @term 
				FILTER cvt.graph_id == cv._id
				FILTER cvt.deprecated == false
				RETURN cvt._id
	`
)
