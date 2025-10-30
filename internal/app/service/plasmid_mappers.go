package service

import (
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

	if m.PlasmidProperties != nil {
		attr.ImageMap = m.PlasmidProperties.ImageMap
		attr.Sequence = m.PlasmidProperties.Sequence
		attr.Name = m.PlasmidProperties.Name
		attr.DictyPlasmidProperty = m.PlasmidProperties.DictyPlasmidProperty
	}

	return attr
}
