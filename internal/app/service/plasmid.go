package service

import (
	"context"
	"fmt"

	"github.com/dictyBase/aphgrpc"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/collection"
	"github.com/dictyBase/modware-stock/internal/model"
)

// CreatePlasmid handles the creation of a new plasmid
func (s *StockService) CreatePlasmid(
	ctx context.Context,
	r *stock.NewPlasmid,
) (*stock.Plasmid, error) {
	plasmid := &stock.Plasmid{}
	if err := r.Validate(); err != nil {
		return plasmid, aphgrpc.HandleInvalidParamError(ctx, err)
	}
	if len(r.Data.Attributes.DictyPlasmidProperty) == 0 {
		r.Data.Attributes.DictyPlasmidProperty = s.Params["plasmid_term"]
	}
	stockDoc, err := s.repo.AddPlasmid(r)
	if err != nil {
		return plasmid, aphgrpc.HandleInsertError(ctx, err)
	}
	plasmid.Data = makePlasmidData(stockDoc)
	err = s.publisher.PublishPlasmid(s.Topics["stockCreate"], plasmid)
	if err != nil {
		return plasmid, aphgrpc.HandleMessagingPubError(ctx, err)
	}
	return plasmid, nil
}

// GetPlasmid handles getting a plasmid by its ID
func (s *StockService) GetPlasmid(
	ctx context.Context,
	r *stock.StockId,
) (*stock.Plasmid, error) {
	plasmid := &stock.Plasmid{}
	if err := r.Validate(); err != nil {
		return plasmid, aphgrpc.HandleInvalidParamError(ctx, err)
	}
	stockDoc, err := s.repo.GetPlasmid(r.Id)
	if err != nil {
		return plasmid, aphgrpc.HandleGetError(ctx, err)
	}
	if stockDoc.NotFound {
		return plasmid,
			aphgrpc.HandleNotFoundError(
				ctx,
				fmt.Errorf("could not find plasmid with ID %s", r.Id),
			)
	}
	plasmid.Data = makePlasmidData(stockDoc)
	return plasmid, nil
}

// LoadPlasmid loads plasmids with existing IDs into the database
func (s *StockService) LoadPlasmid(
	ctx context.Context,
	r *stock.ExistingPlasmid,
) (*stock.Plasmid, error) {
	plasmid := &stock.Plasmid{}
	if err := r.Validate(); err != nil {
		return plasmid, aphgrpc.HandleInvalidParamError(ctx, err)
	}
	if len(r.Data.Attributes.DictyPlasmidProperty) == 0 {
		r.Data.Attributes.DictyPlasmidProperty = s.Params["plasmid_term"]
	}
	id := r.Data.Id
	stockDoc, err := s.repo.LoadPlasmid(id, r)
	if err != nil {
		return plasmid, aphgrpc.HandleInsertError(ctx, err)
	}
	plasmid.Data = makePlasmidData(stockDoc)
	// include ontology property if available
	if stockDoc.PlasmidProperties != nil {
		plasmid.Data.Attributes.DictyPlasmidProperty = stockDoc.PlasmidProperties.DictyPlasmidProperty
	}
	err = s.publisher.PublishPlasmid(s.Topics["stockCreate"], plasmid)
	if err != nil {
		return plasmid, aphgrpc.HandleMessagingPubError(ctx, err)
	}
	return plasmid, nil
}

// UpdatePlasmid handles updating an existing plasmid
func (s *StockService) UpdatePlasmid(
	ctx context.Context,
	r *stock.PlasmidUpdate,
) (*stock.Plasmid, error) {
	plasmid := &stock.Plasmid{}
	if err := r.Validate(); err != nil {
		return plasmid, aphgrpc.HandleInvalidParamError(ctx, err)
	}
	stockDoc, err := s.repo.EditPlasmid(r)
	if err != nil {
		return plasmid, aphgrpc.HandleUpdateError(ctx, err)
	}
	if stockDoc.NotFound {
		return plasmid,
			aphgrpc.HandleNotFoundError(
				ctx,
				fmt.Errorf("could not find plasmid with ID %s", stockDoc.ID),
			)
	}
	plasmid.Data = makePlasmidData(stockDoc)
	err = s.publisher.PublishPlasmid(s.Topics["stockUpdate"], plasmid)
	if err != nil {
		return plasmid, aphgrpc.HandleMessagingPubError(ctx, err)
	}
	return plasmid, nil
}

// ListPlasmids lists all existing plasmids
func (s *StockService) ListPlasmids(
	ctx context.Context,
	param *stock.StockParameters,
) (*stock.PlasmidCollection, error) {
	limit := limitVal(param.Limit)
	pc := &stock.PlasmidCollection{Meta: &stock.Meta{Limit: limit}}
	mc, err := stockModelList(&modelListParams{
		ctx:         ctx,
		stockParams: param,
		limit:       limit,
		fn:          s.repo.ListPlasmids,
	})
	if err != nil {
		return pc, err
	}
	pdata := plasmidModelToCollectionSlice(mc)
	if len(pdata) < int(limit)-2 { // fewer results than limit
		pc.Data = pdata
		pc.Meta.Total = int64(len(pdata))
		return pc, nil
	}
	pc.Data = pdata[:len(pdata)-1]
	pc.Meta.NextCursor = genNextCursorVal(
		pdata[len(pdata)-1].Attributes.CreatedAt,
	)
	pc.Meta.Total = int64(len(pdata))
	return pc, nil
}

func makePlasmidData(m *model.StockDoc) *stock.Plasmid_Data {
	return &stock.Plasmid_Data{
		Type:       "plasmid",
		Id:         m.Key,
		Attributes: makePlasmidAttr(m),
	}
}

func plasmidModelToCollectionSlice(
	mc []*model.StockDoc,
) []*stock.PlasmidCollection_Data {
	return collection.Map(mc, func(m *model.StockDoc) *stock.PlasmidCollection_Data {
		return &stock.PlasmidCollection_Data{
			Type:       "plasmid",
			Id:         m.Key,
			Attributes: makePlasmidAttr(m),
		}
	})
}

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
