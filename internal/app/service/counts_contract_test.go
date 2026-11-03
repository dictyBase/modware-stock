package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/require"
	empty "google.golang.org/protobuf/types/known/emptypb"
)

type contractCountServer struct {
	stock.UnimplementedStockServiceServer
}

func (contractCountServer) CountStocks(
	_ context.Context,
	_ *empty.Empty,
) (*stock.StockCounts, error) {
	return &stock.StockCounts{
		Strains: &stock.StockEntityCounts{
			Total: 12,
			Subtypes: []*stock.StockSubtypeCount{
				{Name: subtypeRegularName, Count: 5},
				{Name: subtypeGwdiName, Count: 7},
				{Name: subtypeBacterialName, Count: 0},
			},
		},
		Plasmids: &stock.StockEntityCounts{
			Total: 4,
			Subtypes: []*stock.StockSubtypeCount{
				{Name: subtypeRegularName, Count: 1},
				{Name: subtypeGoldenBraidName, Count: 3},
			},
		},
	}, nil
}

func TestStockCountsProtoContract(t *testing.T) {
	assert := require.New(t)
	mux := runtime.NewServeMux()
	err := stock.RegisterStockServiceHandlerServer(
		context.Background(),
		mux,
		contractCountServer{},
	)
	assert.NoError(err)

	req := httptest.NewRequest(http.MethodGet, "/v1/stocks/counts", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(http.StatusOK, rec.Code)
	assert.JSONEq(
		`{
		  "strains": {
		    "total": "12",
		    "subtypes": [
		      {"name": "regular", "count": "5"},
		      {"name": "gwdi", "count": "7"},
		      {"name": "bacterial", "count": "0"}
		    ]
		  },
		  "plasmids": {
		    "total": "4",
		    "subtypes": [
		      {"name": "regular", "count": "1"},
		      {"name": "goldenbraid", "count": "3"}
		    ]
		  }
		}`,
		rec.Body.String(),
	)
}
