package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/dictyBase/aphgrpc"
	"github.com/dictyBase/arangomanager/query"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/model"
	"github.com/dictyBase/modware-stock/internal/repository/arangodb"
)

// GetStrain handles getting a strain by its ID
func (s *StockService) GetStrain(ctx context.Context, r *stock.StockId) (*stock.Strain, error) {
	st := &stock.Strain{}
	if err := r.Validate(); err != nil {
		return st, aphgrpc.HandleInvalidParamError(ctx, err)
	}
	m, err := s.repo.GetStrain(r.Id)
	if err != nil {
		return st, aphgrpc.HandleGetError(ctx, err)
	}
	if m.NotFound {
		return st, aphgrpc.HandleNotFoundError(ctx, fmt.Errorf("could not find strain with ID %s", r.Id))
	}
	st.Data = s.makeStrainData(m)
	return st, nil
}

// LoadStock loads strains with existing IDs into the database
func (s *StockService) LoadStrain(ctx context.Context, r *stock.ExistingStrain) (*stock.Strain, error) {
	st := &stock.Strain{}
	if err := r.Validate(); err != nil {
		return st, aphgrpc.HandleInvalidParamError(ctx, err)
	}
	if len(r.Data.Attributes.DictyStrainProperty) == 0 {
		r.Data.Attributes.DictyStrainProperty = s.Params["strain_term"]
	}
	id := r.Data.Id
	m, err := s.repo.LoadStrain(id, r)
	if err != nil {
		return st, aphgrpc.HandleInsertError(ctx, err)
	}
	st.Data = s.makeStrainData(m)
	s.publisher.PublishStrain(s.Topics["stockCreate"], st)
	return st, nil
}

// CreateStrain handles the creation of a new strain
func (s *StockService) CreateStrain(ctx context.Context, r *stock.NewStrain) (*stock.Strain, error) {
	st := &stock.Strain{}
	if err := r.Validate(); err != nil {
		return st, aphgrpc.HandleInvalidParamError(ctx, err)
	}
	if len(r.Data.Attributes.DictyStrainProperty) == 0 {
		r.Data.Attributes.DictyStrainProperty = s.Params["strain_term"]
	}
	m, err := s.repo.AddStrain(r)
	if err != nil {
		return st, aphgrpc.HandleInsertError(ctx, err)
	}
	st.Data = s.makeStrainData(m)
	s.publisher.PublishStrain(s.Topics["stockCreate"], st)
	return st, nil
}

// UpdateStrain handles updating an existing strain
func (s *StockService) UpdateStrain(ctx context.Context, r *stock.StrainUpdate) (*stock.Strain, error) {
	st := &stock.Strain{}
	if err := r.Validate(); err != nil {
		return st, aphgrpc.HandleInvalidParamError(ctx, err)
	}
	m, err := s.repo.EditStrain(r)
	if err != nil {
		return st, aphgrpc.HandleUpdateError(ctx, err)
	}
	if m.NotFound {
		return st,
			aphgrpc.HandleNotFoundError(
				ctx,
				fmt.Errorf("could not find strain with ID %s", m.ID),
			)
	}
	st.Data = s.makeStrainData(m)
	st.Data.Attributes.DictyStrainProperty = ""
	s.publisher.PublishStrain(s.Topics["stockUpdate"], st)
	return st, nil
}

// ListStrainsByIds gets a list of strains from a list of strain identifiers
func (s *StockService) ListStrainsByIds(ctx context.Context, r *stock.StockIdList) (*stock.StrainList, error) {
	sl := &stock.StrainList{}
	if err := r.Validate(); err != nil {
		return sl, aphgrpc.HandleInvalidParamError(ctx, err)
	}
	mc, err := s.repo.ListStrainsByIds(r)
	if err != nil {
		return sl, aphgrpc.HandleGetError(ctx, err)
	}
	if len(mc) == 0 {
		return sl,
			aphgrpc.HandleNotFoundError(
				ctx,
				fmt.Errorf("could not find any strains"),
			)
	}
	sl.Data = strainModelToListSlice(mc)
	return sl, nil
}

// ListStrains lists all existing strains
func (s *StockService) ListStrains(ctx context.Context, r *stock.StockParameters) (*stock.StrainCollection, error) {
	limit := int64(10)
	if r.Limit > 10 {
		limit = r.Limit
	}
	sc := &stock.StrainCollection{Meta: &stock.Meta{Limit: limit}}
	mc, err := s.strainModelList(ctx, r, limit)
	if err != nil {
		return sc, err
	}
	sdata := strainModelToCollectionSlice(mc)
	if len(sdata) < int(limit)-2 { // fewer results than limit
		sc.Data = sdata
		sc.Meta.Total = int64(len(sdata))
		return sc, nil
	}
	sc.Data = sdata[:len(sdata)-1]
	sc.Meta.NextCursor = genNextCursorVal(sdata[len(sdata)-1].Attributes.CreatedAt)
	sc.Meta.Total = int64(len(sdata))
	return sc, nil
}

func (s *StockService) makeStrainData(m *model.StockDoc) *stock.Strain_Data {
	return &stock.Strain_Data{
		Type: "strain",
		Id:   m.Key,
		Attributes: &stock.StrainAttributes{
			CreatedAt:           aphgrpc.TimestampProto(m.CreatedAt),
			UpdatedAt:           aphgrpc.TimestampProto(m.UpdatedAt),
			CreatedBy:           m.CreatedBy,
			UpdatedBy:           m.UpdatedBy,
			Summary:             m.Summary,
			EditableSummary:     m.EditableSummary,
			Depositor:           m.Depositor,
			Genes:               m.Genes,
			Dbxrefs:             m.Dbxrefs,
			Publications:        m.Publications,
			Label:               m.StrainProperties.Label,
			Species:             m.StrainProperties.Species,
			Plasmid:             m.StrainProperties.Plasmid,
			Parent:              m.StrainProperties.Parent,
			Names:               m.StrainProperties.Names,
			DictyStrainProperty: m.StrainProperties.DictyStrainProperty,
		},
	}
}

func (s *StockService) strainModelList(ctx context.Context, r *stock.StockParameters, limit int64) ([]*model.StockDoc, error) {
	astmt, err := strainAQLStatement(r.Filter)
	if err != nil {
		return []*model.StockDoc{}, aphgrpc.HandleInvalidParamError(ctx, err)
	}
	mc, err := s.repo.ListStrains(&stock.StockParameters{
		Cursor: r.Cursor,
		Limit:  limit,
		Filter: astmt,
	})
	if err != nil {
		return mc, aphgrpc.HandleGetError(ctx, err)
	}
	if len(mc) == 0 {
		return mc,
			aphgrpc.HandleNotFoundError(
				ctx, errors.New("could not find any strains"),
			)
	}
	return mc, nil
}

func strainModelToCollectionSlice(mc []*model.StockDoc) []*stock.StrainCollection_Data {
	var sdata []*stock.StrainCollection_Data
	for _, m := range mc {
		sdata = append(sdata, &stock.StrainCollection_Data{
			Type: "strain",
			Id:   m.Key,
			Attributes: &stock.StrainAttributes{
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
				Label:           m.StrainProperties.Label,
				Species:         m.StrainProperties.Species,
				Plasmid:         m.StrainProperties.Plasmid,
				Parent:          m.StrainProperties.Parent,
				Names:           m.StrainProperties.Names,
			},
		})
	}
	return sdata
}

func strainModelToListSlice(mc []*model.StockDoc) []*stock.StrainList_Data {
	var sdata []*stock.StrainList_Data
	for _, m := range mc {
		sdata = append(sdata, &stock.StrainList_Data{
			Type: "strain",
			Id:   m.Key,
			Attributes: &stock.StrainAttributes{
				CreatedAt:           aphgrpc.TimestampProto(m.CreatedAt),
				UpdatedAt:           aphgrpc.TimestampProto(m.UpdatedAt),
				CreatedBy:           m.CreatedBy,
				UpdatedBy:           m.UpdatedBy,
				Summary:             m.Summary,
				EditableSummary:     m.EditableSummary,
				Depositor:           m.Depositor,
				Genes:               m.Genes,
				Dbxrefs:             m.Dbxrefs,
				Publications:        m.Publications,
				Label:               m.StrainProperties.Label,
				Species:             m.StrainProperties.Species,
				Plasmid:             m.StrainProperties.Plasmid,
				Parent:              m.StrainProperties.Parent,
				Names:               m.StrainProperties.Names,
				DictyStrainProperty: m.StrainProperties.DictyStrainProperty,
			},
		})
	}
	return sdata
}

func strainAQLStatement(filter string) (string, error) {
	var vert bool
	var astmt string
	p, err := query.ParseFilterString(filter)
	if err != nil {
		return astmt,
			fmt.Errorf("error in parsing filter string %s", err)
	}
	// need to check if filter contains an item found in strain properties
	for _, n := range p {
		if _, ok := stockProp[n.Field]; ok {
			vert = true
			break
		}
	}
	if vert {
		astmt, err := query.GenAQLFilterStatement(&query.StatementParameters{
			Fmap:    arangodb.FMap,
			Filters: p,
			Vert:    "v",
		})
		if err != nil {
			return astmt,
				fmt.Errorf("error in generating AQL statement %s", err)
		}
	} else {
		astmt, err := query.GenAQLFilterStatement(&query.StatementParameters{
			Fmap:    arangodb.FMap,
			Filters: p,
			Doc:     "s",
		})
		if err != nil {
			return astmt,
				fmt.Errorf("error in generating AQL statement %s", err)
		}
	}
	// if the parsed statement is empty FILTER, just return empty string
	if astmt == "FILTER " {
		astmt = ""
	}
	return astmt, nil
}
