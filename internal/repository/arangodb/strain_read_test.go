package arangodb

import (
	"testing"

	"github.com/dictyBase/modware-stock/internal/model"
)

func Test_arangorepository_GetStrain(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		id      string
		want    *model.StockDoc
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO: construct the receiver type.
			var ar arangorepository
			got, gotErr := ar.GetStrain(tt.id)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("GetStrain() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("GetStrain() succeeded unexpectedly")
			}
			// TODO: update the condition below to compare got with tt.want.
			if true {
				t.Errorf("GetStrain() = %v, want %v", got, tt.want)
			}
		})
	}
}
