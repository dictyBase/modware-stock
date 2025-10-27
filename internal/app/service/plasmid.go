package service

import (
	"context"
	"fmt"

	"github.com/dictyBase/aphgrpc"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/model"
)

// CreatePlasmid handles the creation of a new plasmid
func (s *StockService) CreatePlasmid(
	ctx context.Context,
	r *stock.NewPlasmid,
) (*stock.Plasmid, error) {
	st := &stock.Plasmid{}
	if err := r.Validate(); err != nil {
		return st, aphgrpc.HandleInvalidParamError(ctx, err)
	}
	if len(r.Data.Attributes.DictyPlasmidProperty) == 0 {
		r.Data.Attributes.DictyPlasmidProperty = s.Params["plasmid_term"]
	}
	m, err := s.repo.AddPlasmid(r)
	if err != nil {
		return st, aphgrpc.HandleInsertError(ctx, err)
	}
	st.Data = makePlasmidData(m)
	err = s.publisher.PublishPlasmid(s.Topics["stockCreate"], st)
	if err != nil {
		return st, aphgrpc.HandleMessagingPubError(ctx, err)
	}
	return st, nil
}

// GetPlasmid handles getting a plasmid by its ID
func (s *StockService) GetPlasmid(
	ctx context.Context,
	r *stock.StockId,
) (*stock.Plasmid, error) {
	pl := &stock.Plasmid{}
	if err := r.Validate(); err != nil {
		return pl, aphgrpc.HandleInvalidParamError(ctx, err)
	}
	m, err := s.repo.GetPlasmid(r.Id)
	if err != nil {
		return pl, aphgrpc.HandleGetError(ctx, err)
	}
	if m.NotFound {
		return pl,
			aphgrpc.HandleNotFoundError(
				ctx,
				fmt.Errorf("could not find plasmid with ID %s", r.Id),
			)
	}
	pl.Data = makePlasmidData(m)
	return pl, nil
}

// LoadPlasmid loads plasmids with existing IDs into the database
func (s *StockService) LoadPlasmid(
	ctx context.Context,
	r *stock.ExistingPlasmid,
) (*stock.Plasmid, error) {
	pl := &stock.Plasmid{}
	if err := r.Validate(); err != nil {
		return pl, aphgrpc.HandleInvalidParamError(ctx, err)
	}
	if len(r.Data.Attributes.DictyPlasmidProperty) == 0 {
		r.Data.Attributes.DictyPlasmidProperty = s.Params["plasmid_term"]
	}
	id := r.Data.Id
	m, err := s.repo.LoadPlasmid(id, r)
	if err != nil {
		return pl, aphgrpc.HandleInsertError(ctx, err)
	}
	pl.Data = makePlasmidData(m)
	// include ontology property if available
	if m.PlasmidProperties != nil {
		pl.Data.Attributes.DictyPlasmidProperty = m.PlasmidProperties.DictyPlasmidProperty
	}
	err = s.publisher.PublishPlasmid(s.Topics["stockCreate"], pl)
	if err != nil {
		return pl, aphgrpc.HandleMessagingPubError(ctx, err)
	}
	return pl, nil
}

// UpdatePlasmid handles updating an existing plasmid
func (s *StockService) UpdatePlasmid(
	ctx context.Context,
	r *stock.PlasmidUpdate,
) (*stock.Plasmid, error) {
	pl := &stock.Plasmid{}
	if err := r.Validate(); err != nil {
		return pl, aphgrpc.HandleInvalidParamError(ctx, err)
	}
	m, err := s.repo.EditPlasmid(r)
	if err != nil {
		return pl, aphgrpc.HandleUpdateError(ctx, err)
	}
	if m.NotFound {
		return pl,
			aphgrpc.HandleNotFoundError(
				ctx,
				fmt.Errorf("could not find plasmid with ID %s", m.ID),
			)
	}
	pl.Data = makePlasmidData(m)
	err = s.publisher.PublishPlasmid(s.Topics["stockUpdate"], pl)
	if err != nil {
		return pl, aphgrpc.HandleMessagingPubError(ctx, err)
	}
	return pl, nil
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
	var pdata []*stock.PlasmidCollection_Data
	for _, m := range mc {
		pdata = append(pdata, &stock.PlasmidCollection_Data{
			Type:       "plasmid",
			Id:         m.Key,
			Attributes: makePlasmidAttr(m),
		})
	}
	return pdata
}

func makePlasmidAttr(m *model.StockDoc) *stock.PlasmidAttributes {
	return &stock.PlasmidAttributes{
		CreatedAt:            aphgrpc.TimestampProto(m.CreatedAt),
		UpdatedAt:            aphgrpc.TimestampProto(m.UpdatedAt),
		CreatedBy:            m.CreatedBy,
		UpdatedBy:            m.UpdatedBy,
		Summary:              m.Summary,
		EditableSummary:      m.EditableSummary,
		Depositor:            m.Depositor,
		Genes:                m.Genes,
		Dbxrefs:              m.Dbxrefs,
		Publications:         m.Publications,
		ImageMap:             m.PlasmidProperties.ImageMap,
		Sequence:             m.PlasmidProperties.Sequence,
		Name:                 m.PlasmidProperties.Name,
		DictyPlasmidProperty: m.PlasmidProperties.DictyPlasmidProperty,
	}
}
