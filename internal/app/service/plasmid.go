package service

import (
	"context"
	"fmt"

	"github.com/dictyBase/aphgrpc"
	"github.com/dictyBase/arangomanager/query"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/repository/arangodb"
)

// CreatePlasmid handles the creation of a new plasmid
func (s *StockService) CreatePlasmid(ctx context.Context, r *stock.NewPlasmid) (*stock.Plasmid, error) {
	st := &stock.Plasmid{}
	if err := r.Validate(); err != nil {
		return st, aphgrpc.HandleInvalidParamError(ctx, err)
	}

	m, err := s.repo.AddPlasmid(r)
	if err != nil {
		return st, aphgrpc.HandleInsertError(ctx, err)
	}
	st.Data = &stock.Plasmid_Data{
		Type: "plasmid",
		Id:   m.Key,
		Attributes: &stock.PlasmidAttributes{
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
		},
	}
	s.publisher.PublishPlasmid(s.Topics["stockCreate"], st)
	return st, nil
}

// GetPlasmid handles getting a plasmid by its ID
func (s *StockService) GetPlasmid(ctx context.Context, r *stock.StockId) (*stock.Plasmid, error) {
	st := &stock.Plasmid{}
	if err := r.Validate(); err != nil {
		return st, aphgrpc.HandleInvalidParamError(ctx, err)
	}

	m, err := s.repo.GetPlasmid(r.Id)
	if err != nil {
		return st, aphgrpc.HandleGetError(ctx, err)
	}
	if m.NotFound {
		return st, aphgrpc.HandleNotFoundError(ctx, fmt.Errorf("could not find plasmid with ID %s", r.Id))
	}
	st.Data = &stock.Plasmid_Data{
		Type: "plasmid",
		Id:   m.Key,
		Attributes: &stock.PlasmidAttributes{
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
		},
	}
	return st, nil
}

// UpdatePlasmid handles updating an existing plasmid
func (s *StockService) UpdatePlasmid(ctx context.Context, r *stock.PlasmidUpdate) (*stock.Plasmid, error) {
	st := &stock.Plasmid{}
	if err := r.Validate(); err != nil {
		return st, aphgrpc.HandleInvalidParamError(ctx, err)
	}
	m, err := s.repo.EditPlasmid(r)
	if err != nil {
		return st, aphgrpc.HandleUpdateError(ctx, err)
	}
	if m.NotFound {
		return st, aphgrpc.HandleNotFoundError(ctx, fmt.Errorf("could not find plasmid with ID %s", m.ID))
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
	s.publisher.PublishPlasmid(s.Topics["stockUpdate"], st)
	return st, nil
}

// ListPlasmids lists all existing plasmids
func (s *StockService) ListPlasmids(ctx context.Context, r *stock.StockParameters) (*stock.PlasmidCollection, error) {
	pc := &stock.PlasmidCollection{}
	limit := int64(10)
	if r.Limit > 10 {
		limit = r.Limit
	}
	var astmt string
	var vert bool
	if len(r.Filter) > 0 {
		p, err := query.ParseFilterString(r.Filter)
		if err != nil {
			return pc, aphgrpc.HandleInvalidParamError(
				ctx,
				fmt.Errorf("error in parsing filter string"),
			)
		}
		for _, n := range p {
			if _, ok := stockProp[n.Field]; ok {
				vert = true
				break
			}
		}
		if vert {
			astmt, err = query.GenAQLFilterStatement(&query.StatementParameters{Fmap: arangodb.FMap, Filters: p, Vert: "v"})
			if err != nil {
				return pc, aphgrpc.HandleInvalidParamError(
					ctx,
					fmt.Errorf("error in generating AQL statement"),
				)
			}
		} else {
			astmt, err = query.GenAQLFilterStatement(&query.StatementParameters{Fmap: arangodb.FMap, Filters: p, Doc: "s"})
			if err != nil {
				return pc, aphgrpc.HandleInvalidParamError(
					ctx,
					fmt.Errorf("error in generating AQL statement"),
				)
			}
		}
		// if the parsed statement is empty FILTER, just return empty string
		if astmt == "FILTER " {
			astmt = ""
		}
	}
	mc, err := s.repo.ListPlasmids(&stock.StockParameters{Cursor: r.Cursor, Limit: limit, Filter: astmt})
	if err != nil {
		return pc, aphgrpc.HandleGetError(ctx, err)
	}
	if len(mc) == 0 {
		return pc, aphgrpc.HandleNotFoundError(ctx, fmt.Errorf("could not find any plasmids"))
	}
	var pdata []*stock.PlasmidCollection_Data
	for _, m := range mc {
		pdata = append(pdata, &stock.PlasmidCollection_Data{
			Type: "plasmid",
			Id:   m.Key,
			Attributes: &stock.PlasmidAttributes{
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
			},
		})
	}
	if len(pdata) < int(limit)-2 { // fewer results than limit
		pc.Data = pdata
		pc.Meta = &stock.Meta{
			Limit: limit,
			Total: int64(len(pdata)),
		}
		return pc, nil
	}
	pc.Data = pdata[:len(pdata)-1]
	pc.Meta = &stock.Meta{
		Limit:      limit,
		NextCursor: genNextCursorVal(pdata[len(pdata)-1].Attributes.CreatedAt),
		Total:      int64(len(pdata)),
	}
	return pc, nil
}

// LoadPlasmid loads plasmids with existing IDs into the database
func (s *StockService) LoadPlasmid(ctx context.Context, r *stock.ExistingPlasmid) (*stock.Plasmid, error) {
	st := &stock.Plasmid{}
	if err := r.Validate(); err != nil {
		return st, aphgrpc.HandleInvalidParamError(ctx, err)
	}
	id := r.Data.Id

	m, err := s.repo.LoadPlasmid(id, r)
	if err != nil {
		return st, aphgrpc.HandleInsertError(ctx, err)
	}
	st.Data = &stock.Plasmid_Data{
		Type: "plasmid",
		Id:   m.Key,
		Attributes: &stock.PlasmidAttributes{
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
		},
	}

	s.publisher.PublishPlasmid(s.Topics["stockCreate"], st)
	return st, nil
}
