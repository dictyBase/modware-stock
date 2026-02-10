package arangodb

// copyCollectionParamsWithOverride creates a copy of CollectionParams with modifications
func copyCollectionParamsWithOverride(original *CollectionParams, modify func(*CollectionParams)) *CollectionParams {
	copy := &CollectionParams{
		Stock:              original.Stock,
		StockProp:          original.StockProp,
		StockType:          original.StockType,
		StockKeyGenerator:  original.StockKeyGenerator,
		ParentStrain:       original.ParentStrain,
		StockTerm:          original.StockTerm,
		StockPropTypeGraph: original.StockPropTypeGraph,
		Strain2ParentGraph: original.Strain2ParentGraph,
		StockOntoGraph:     original.StockOntoGraph,
		KeyOffset:          original.KeyOffset,
		StrainOntology:     original.StrainOntology,
		PlasmidOntology:    original.PlasmidOntology,
	}
	modify(copy)
	return copy
}
