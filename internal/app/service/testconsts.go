package service

const (
	// Test fixture user emails shared by the service test helpers.
	testUserEmail       = "testuser@dictybase.org"
	testLoadUserEmail   = "loaduser@dictybase.org"
	testUpdateUserEmail = "updateuser@dictybase.org"

	// Test fixture field values shared by the service test helpers.
	testStrainLabel      = "yS13"
	testStrainTag        = "general strain"
	testUpdatedSummary   = "Updated summary"
	testDepositorFilter  = "depositor===John Doe"
	testTagFilter        = "tag===Gateway vector"
	testInvalidIDFormat  = "invalid-id-format"
	testSpecies          = "Dictyostelium discoideum"
	testRemiSeqProperty  = "REMI-seq"
	testMissingStrainID  = "DBS9999999"
	testInvalidSubstring = "invalid"
	// testStockID is the fixture strain identifier.
	testStockID = "DBS0236126"
	// testPlasmidID is the fixture plasmid identifier.
	testPlasmidID = "DBP0000027"
	// testPlasmidName is the fixture plasmid name.
	testPlasmidName = "pDM304"

	// Subtype group names expected by the CountStocks tests. The proto
	// contract and the AQL statement pin the same names.
	subtypeRegularName     = "regular"
	subtypeGwdiName        = "gwdi"
	subtypeBacterialName   = "bacterial"
	subtypeGoldenBraidName = "goldenbraid"
)
