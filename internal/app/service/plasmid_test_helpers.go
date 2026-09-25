package service

import (
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	// testGatewayVector is the test value for DictyPlasmidProperty in Gateway vector tests.
	testGatewayVector = "Gateway vector"
)

// newTestPlasmid provides a consistent *stock.NewPlasmid for testing.
func newTestPlasmid() *stock.NewPlasmid {
	return &stock.NewPlasmid{
		Data: &stock.NewPlasmid_Data{
			Type: stockTypePlasmid,
			Attributes: &stock.NewPlasmidAttributes{
				CreatedBy:       testUserEmail,
				UpdatedBy:       testUserEmail,
				Summary:         "Test summary for plasmid",
				EditableSummary: "Editable summary",
				Depositor:       "John Doe",
				Genes:           []string{"gene1", "gene2"},
				Dbxrefs:         []string{"dbxref1", "dbxref2"},
				Publications:    []string{"pub1", "pub2"},
				ImageMap:        "https://example.com/image.png",
				Sequence:        "ATCGATCGATCG",
				Name:            "pDV101",
			},
		},
	}
}

// newExistingPlasmid creates a test ExistingPlasmid request.
func newExistingPlasmid() *stock.ExistingPlasmid {
	return &stock.ExistingPlasmid{
		Data: &stock.ExistingPlasmid_Data{
			Type: stockTypePlasmid,
			Id:   "DBP0000001",
			Attributes: &stock.ExistingPlasmidAttributes{
				CreatedBy:       testLoadUserEmail,
				UpdatedBy:       testLoadUserEmail,
				CreatedAt:       timestamppb.Now(),
				UpdatedAt:       timestamppb.Now(),
				Summary:         "Loaded plasmid summary",
				EditableSummary: "Editable loaded summary",
				Depositor:       "Load User",
				Genes:           []string{"geneA", "geneB"},
				Dbxrefs:         []string{"dbxrefA"},
				Publications:    []string{"pubA"},
				ImageMap:        "https://example.com/loaded.png",
				Sequence:        "GCTAGCTAGCTA",
				Name:            "pLoadTest",
			},
		},
	}
}

// newPlasmidUpdate creates a test PlasmidUpdate request.
func newPlasmidUpdate(plasmidID string) *stock.PlasmidUpdate {
	return &stock.PlasmidUpdate{
		Data: &stock.PlasmidUpdate_Data{
			Type: stockTypePlasmid,
			Id:   plasmidID,
			Attributes: &stock.PlasmidUpdateAttributes{
				UpdatedBy:       testUpdateUserEmail,
				Summary:         testUpdatedSummary,
				EditableSummary: "Updated editable summary",
				Genes:           []string{"geneX", "geneY"},
				Dbxrefs:         []string{"dbxrefX"},
			},
		},
	}
}
