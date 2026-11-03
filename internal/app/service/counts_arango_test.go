package service

import (
	"context"
	"testing"

	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	empty "google.golang.org/protobuf/types/known/emptypb"
)

func TestCountStocksEndToEnd(t *testing.T) {
	client, assert := newFullSearchArangoEnv(t)
	ctx := context.Background()

	_, err := client.CreateStrain(ctx, &stock.NewStrain{
		Data: &stock.NewStrain_Data{
			Type: stockTypeStrain,
			Attributes: &stock.NewStrainAttributes{
				CreatedBy:           testUserEmail,
				UpdatedBy:           testUserEmail,
				Depositor:           testUserEmail,
				Summary:             "general strain summary",
				EditableSummary:     "general strain summary",
				Label:               testStrainLabel,
				Species:             testSpecies,
				DictyStrainProperty: testStrainTag,
			},
		},
	})
	assert.NoError(err)

	_, err = client.CreateStrain(ctx, &stock.NewStrain{
		Data: &stock.NewStrain_Data{
			Type: stockTypeStrain,
			Attributes: &stock.NewStrainAttributes{
				CreatedBy:           testUserEmail,
				UpdatedBy:           testUserEmail,
				Depositor:           testUserEmail,
				Summary:             "gwdi strain summary",
				EditableSummary:     "gwdi strain summary",
				Label:               "gwdi-1",
				Species:             testSpecies,
				DictyStrainProperty: testRemiSeqProperty,
			},
		},
	})
	assert.NoError(err)

	_, err = client.CreatePlasmid(ctx, &stock.NewPlasmid{
		Data: &stock.NewPlasmid_Data{
			Type: stockTypePlasmid,
			Attributes: &stock.NewPlasmidAttributes{
				CreatedBy:            testUserEmail,
				UpdatedBy:            testUserEmail,
				Depositor:            testUserEmail,
				Name:                 "pReg1",
				DictyPlasmidProperty: "vector",
			},
		},
	})
	assert.NoError(err)

	_, err = client.CreatePlasmid(ctx, &stock.NewPlasmid{
		Data: &stock.NewPlasmid_Data{
			Type: stockTypePlasmid,
			Attributes: &stock.NewPlasmidAttributes{
				CreatedBy:            testUserEmail,
				UpdatedBy:            testUserEmail,
				Depositor:            testUserEmail,
				Name:                 "pGB1",
				DictyPlasmidProperty: "GB vector",
			},
		},
	})
	assert.NoError(err)

	res, err := client.CountStocks(ctx, &empty.Empty{})
	assert.NoError(err)
	assert.Equal(int64(2), res.GetStrains().GetTotal())
	assert.Equal(
		[]*stock.StockSubtypeCount{
			{Name: subtypeRegularName, Count: 1},
			{Name: subtypeGwdiName, Count: 1},
			{Name: subtypeBacterialName, Count: 0},
		},
		res.GetStrains().GetSubtypes(),
	)
	assert.Equal(int64(2), res.GetPlasmids().GetTotal())
	assert.Equal(
		[]*stock.StockSubtypeCount{
			{Name: subtypeRegularName, Count: 1},
			{Name: subtypeGoldenBraidName, Count: 1},
		},
		res.GetPlasmids().GetSubtypes(),
	)
}
