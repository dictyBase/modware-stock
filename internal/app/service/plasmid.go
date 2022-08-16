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
	st := &stock.Plasmid{}
	if err := r.Validate(); err != nil {
		return st, aphgrpc.HandleInvalidParamError(ctx, err)
	}

	m, err := s.repo.GetPlasmid(r.Id)
	if err != nil {
		return st, aphgrpc.HandleGetError(ctx, err)
	}
	if m.NotFound {
		return st,
			aphgrpc.HandleNotFoundError(
				ctx,
				fmt.Errorf("could not find plasmid with ID %s", r.Id),
			)
	}
	st.Data = makePlasmidData(m)
	return st, nil
}

// UpdatePlasmid handles updating an existing plasmid
func (s *StockService) UpdatePlasmid(
	ctx context.Context,
	r *stock.PlasmidUpdate,
) (*stock.Plasmid, error) {
	st := &stock.Plasmid{}
	if err := r.Validate(); err != nil {
		return st, aphgrpc.HandleInvalidParamError(ctx, err)
	}
	m, err := s.repo.EditPlasmid(r)
	if err != nil {
		return st, aphgrpc.HandleUpdateError(ctx, err)
	}
	if m.NotFound {
		return st, aphgrpc.HandleNotFoundError(
			ctx,
			fmt.Errorf("could not find plasmid with ID %s", m.ID),
		)
	}
	st.Data = &stock.Plasmid_Data{
		Type: "plasmid",
		Id:   m.Key,
		Attributes: &stock.PlasmidAttributes{
			UpdatedBy:       m.UpdatedBy,
			Summary:         m.Summary,
			EditableSummary: m.EditableSummary,
			Depositor:       m.Depositor,
			Genes:           m.Genes,
			Dbxrefs:         m.Dbxrefs,
			Publications:    m.Publications,
			ImageMap:        m.PlasmidProperties.ImageMap,
			Sequence:        m.PlasmidProperties.Sequence,
			Name:            m.PlasmidProperties.Name,
		},
	}
	err = s.publisher.PublishPlasmid(s.Topics["stockUpdate"], st)
	if err != nil {
		return st, aphgrpc.HandleMessagingPubError(ctx, err)
	}
	return st, nil
}

// ListPlasmids lists all existing plasmids
func (s *StockService) ListPlasmids(
	ctx context.Context,
	r *stock.StockParameters,
) (*stock.PlasmidCollection, error) {
	limit := limitVal(r.Limit)
	pc := &stock.PlasmidCollection{Meta: &stock.Meta{Limit: limit}}
	mc, err := stockModelList(&modelListParams{
		ctx:         ctx,
		stockParams: r,
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
		pdata[len(pdata)-1].Attributes.CreatedAt.String(),
	)
	pc.Meta.Total = int64(len(pdata))
	return pc, nil
}

// LoadPlasmid loads plasmids with existing IDs into the database
func (s *StockService) LoadPlasmid(
	ctx context.Context,
	r *stock.ExistingPlasmid,
) (*stock.Plasmid, error) {
	st := &stock.Plasmid{}
	if err := r.Validate(); err != nil {
		return st, aphgrpc.HandleInvalidParamError(ctx, err)
	}
	id := r.Data.Id

	m, err := s.repo.LoadPlasmid(id, r)
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
		ImageMap:        m.PlasmidProperties.ImageMap,
		Sequence:        m.PlasmidProperties.Sequence,
		Name:            m.PlasmidProperties.Name,
	}
}
