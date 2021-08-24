package statement

const (
	strainExistTermQ = `
		FOR cv IN @@cv_collection
			FOR cvt IN @@cvterm_collection
				FILTER cv.metadata.namespace == @ontology
				FILTER cvt.label == @term 
				FILTER cvt.graph_id == cv._id
				FILTER cvt.deprecated == false
				RETURN cvt._id
	`
)
