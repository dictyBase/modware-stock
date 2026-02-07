package service

import (
	F "github.com/IBM/fp-go/function"
	O "github.com/IBM/fp-go/option"
	"github.com/dictyBase/aphgrpc"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/collection"
	"github.com/dictyBase/modware-stock/internal/model"
)

// makePlasmidData transforms a stock document model into plasmid data
func makePlasmidData(m *model.StockDoc) *stock.Plasmid_Data {
	return &stock.Plasmid_Data{
		Type:       "plasmid",
		Id:         m.Key,
		Attributes: makePlasmidAttr(m),
	}
}

// plasmidModelToCollectionSlice transforms stock document models into collection data slice
func plasmidModelToCollectionSlice(
	mc []*model.StockDoc,
) []*stock.PlasmidCollection_Data {
	return collection.Map(
		mc,
		func(m *model.StockDoc) *stock.PlasmidCollection_Data {
			return &stock.PlasmidCollection_Data{
				Type:       "plasmid",
				Id:         m.Key,
				Attributes: makePlasmidAttr(m),
			}
		},
	)
}

// makePlasmidAttr creates plasmid attributes from stock document model
func makePlasmidAttr(m *model.StockDoc) *stock.PlasmidAttributes {
	attr := &stock.PlasmidAttributes{
		CreatedAt:       aphgrpc.TimestampProto(m.CreatedAt),
		UpdatedAt:       aphgrpc.TimestampProto(m.UpdatedAt),
		CreatedBy:       m.CreatedBy,
		UpdatedBy:       m.UpdatedBy,
		Summary:         m.Summary,
		EditableSummary: m.EditableSummary,
		Depositor:       m.Depositor,
		Genes:           m.Genes,
		Dbxrefs:         m.Dbxrefs,
		Publications:    m.Publications,
	}
	return F.Pipe1(
		O.FromNillable(m.PlasmidProperties),
		O.Fold(
			F.Constant(attr),
			func(props *model.PlasmidProperties) *stock.PlasmidAttributes {
				attr.ImageMap = props.ImageMap
				attr.Sequence = props.Sequence
				attr.Name = props.Name
				attr.DictyPlasmidProperty = props.DictyPlasmidProperty
				return attr
			},
		),
	)
}
