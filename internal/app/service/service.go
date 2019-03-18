package service

import (
	"context"
	"fmt"
	"strconv"

	"github.com/dictyBase/modware-stock/internal/repository/arangodb"

	"github.com/dictyBase/apihelpers/aphgrpc"
	"github.com/dictyBase/arangomanager/query"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/message"
	"github.com/dictyBase/modware-stock/internal/repository"
	"github.com/golang/protobuf/ptypes/empty"
)

// StockService is the container for managing stock service
// definition
type StockService struct {
	*aphgrpc.Service
	repo      repository.StockRepository
	publisher message.Publisher
}

func defaultOptions() *aphgrpc.ServiceOptions {
	return &aphgrpc.ServiceOptions{Resource: "stock"}
}

// NewStockService is the constructor for creating a new instance of StockService
func NewStockService(repo repository.StockRepository, pub message.Publisher, opt ...aphgrpc.Option) *StockService {
	s := defaultOptions()
	for _, optfn := range opt {
		optfn(s)
	}
	srv := &aphgrpc.Service{}
	aphgrpc.AssignFieldsToStructs(s, srv)
	return &StockService{
		Service:   srv,
		repo:      repo,
		publisher: pub,
	}
}

// GetStock handles getting a stock by its ID
func (s *StockService) GetStock(ctx context.Context, r *stock.StockId) (*stock.Stock, error) {
	st := &stock.Stock{}
	if err := r.Validate(); err != nil {
		return st, aphgrpc.HandleInvalidParamError(ctx, err)
	}
	if len(r.Id) < 3 {
		return st, fmt.Errorf("stock ID %s is not long enough (must begin with DBS or DBP)", r.Id)
	}
	if r.Id[:3] == "DBS" {
		m, err := s.repo.GetStrain(r.Id)
		if err != nil {
			return st, aphgrpc.HandleGetError(ctx, err)
		}
		if m.NotFound {
			return st, aphgrpc.HandleNotFoundError(ctx, fmt.Errorf("could not find strain with ID %s", r.Id))
		}
		st.Data = &stock.Stock_Data{
			Type: s.GetResourceName(),
			Id:   m.Key,
			Attributes: &stock.StockAttributes{
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
				StrainProperties: &stock.StrainProperties{
					SystematicName: m.StrainProperties.SystematicName,
					Label:          m.StrainProperties.Label,
					Species:        m.StrainProperties.Species,
					Plasmid:        m.StrainProperties.Plasmid,
					Parent:         m.StrainProperties.Parent,
					Names:          m.StrainProperties.Names,
				},
			},
		}
	} else if r.Id[:3] == "DBP" {
		m, err := s.repo.GetPlasmid(r.Id)
		if err != nil {
			return st, aphgrpc.HandleGetError(ctx, err)
		}
		if m.NotFound {
			return st, aphgrpc.HandleNotFoundError(ctx, fmt.Errorf("could not find plasmid with ID %s", r.Id))
		}
		st.Data = &stock.Stock_Data{
			Type: s.GetResourceName(),
			Id:   m.Key,
			Attributes: &stock.StockAttributes{
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
				PlasmidProperties: &stock.PlasmidProperties{
					ImageMap: m.PlasmidProperties.ImageMap,
					Sequence: m.PlasmidProperties.Sequence,
				},
			},
		}
	} else {
		return st, fmt.Errorf("stock ID %s is not valid (must begin with DBS or DBP)", r.Id)
	}
	return st, nil
}

// CreateStock handles the creation of a new stock
func (s *StockService) CreateStock(ctx context.Context, r *stock.NewStock) (*stock.Stock, error) {
	st := &stock.Stock{}
	if err := r.Validate(); err != nil {
		return st, aphgrpc.HandleInvalidParamError(ctx, err)
	}
	str := r.Data.GetType()
	if str == "strain" {
		m, err := s.repo.AddStrain(r)
		if err != nil {
			return st, aphgrpc.HandleInsertError(ctx, err)
		}
		st.Data = &stock.Stock_Data{
			Type: s.GetResourceName(),
			Id:   m.Key,
			Attributes: &stock.StockAttributes{
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
				StrainProperties: &stock.StrainProperties{
					SystematicName: m.StrainProperties.SystematicName,
					Label:          m.StrainProperties.Label,
					Species:        m.StrainProperties.Species,
					Plasmid:        m.StrainProperties.Plasmid,
					Parent:         m.StrainProperties.Parent,
					Names:          m.StrainProperties.Names,
				},
			},
		}
	} else {
		m, err := s.repo.AddPlasmid(r)
		if err != nil {
			return st, aphgrpc.HandleInsertError(ctx, err)
		}
		st.Data = &stock.Stock_Data{
			Type: s.GetResourceName(),
			Id:   m.Key,
			Attributes: &stock.StockAttributes{
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
				PlasmidProperties: &stock.PlasmidProperties{
					ImageMap: m.PlasmidProperties.ImageMap,
					Sequence: m.PlasmidProperties.Sequence,
				},
			},
		}
	}
	s.publisher.Publish(s.Topics["stockCreate"], st)
	return st, nil
}

// UpdateStock handles updating an existing stock
func (s *StockService) UpdateStock(ctx context.Context, r *stock.StockUpdate) (*stock.Stock, error) {
	st := &stock.Stock{}
	if err := r.Validate(); err != nil {
		return st, aphgrpc.HandleInvalidParamError(ctx, err)
	}
	str := r.Data.GetType()
	if str == "strain" {
		m, err := s.repo.EditStrain(r)
		if err != nil {
			return st, aphgrpc.HandleUpdateError(ctx, err)
		}
		if m.NotFound {
			return st, aphgrpc.HandleNotFoundError(ctx, fmt.Errorf("could not find strain with ID %s", r.Id))
		}
		st.Data = &stock.Stock_Data{
			Type: s.GetResourceName(),
			Id:   m.Key,
			Attributes: &stock.StockAttributes{
				UpdatedBy:       m.UpdatedBy,
				Summary:         m.Summary,
				EditableSummary: m.EditableSummary,
				Depositor:       m.Depositor,
				Genes:           m.Genes,
				Dbxrefs:         m.Dbxrefs,
				Publications:    m.Publications,
				StrainProperties: &stock.StrainProperties{
					SystematicName: m.StrainProperties.SystematicName,
					Label:          m.StrainProperties.Label,
					Species:        m.StrainProperties.Species,
					Plasmid:        m.StrainProperties.Plasmid,
					Parent:         m.StrainProperties.Parent,
					Names:          m.StrainProperties.Names,
				},
			},
		}
	} else {
		m, err := s.repo.EditPlasmid(r)
		if err != nil {
			return st, aphgrpc.HandleUpdateError(ctx, err)
		}
		if m.NotFound {
			return st, aphgrpc.HandleNotFoundError(ctx, fmt.Errorf("could not find plasmid with ID %s", r.Id))
		}
		st.Data = &stock.Stock_Data{
			Type: s.GetResourceName(),
			Id:   m.Key,
			Attributes: &stock.StockAttributes{
				UpdatedBy:       m.UpdatedBy,
				Summary:         m.Summary,
				EditableSummary: m.EditableSummary,
				Depositor:       m.Depositor,
				Genes:           m.Genes,
				Dbxrefs:         m.Dbxrefs,
				Publications:    m.Publications,
				PlasmidProperties: &stock.PlasmidProperties{
					ImageMap: m.PlasmidProperties.ImageMap,
					Sequence: m.PlasmidProperties.Sequence,
				},
			},
		}
	}
	s.publisher.Publish(s.Topics["stockUpdate"], st)
	return st, nil
}

// ListStrains lists all existing strains
func (s *StockService) ListStrains(ctx context.Context, r *stock.StockParameters) (*stock.StockCollection, error) {
	sc := &stock.StockCollection{}
	c := r.Cursor
	l := r.Limit
	f := r.Filter
	if len(f) > 0 {
		p, err := query.ParseFilterString(f)
		if err != nil {
			return sc, fmt.Errorf("error parsing filter string: %s", err)
		}
		str, err := query.GenAQLFilterStatement(&query.StatementParameters{Fmap: arangodb.FMap, Filters: p, Doc: "s", Vert: "v"})
		if err != nil {
			return sc, fmt.Errorf("error generating AQL filter statement: %s", err)
		}
		// if the parsed statement is empty FILTER, just return empty string
		if str == "FILTER " {
			str = ""
		}
		mc, err := s.repo.ListStrains(&stock.StockParameters{Cursor: c, Limit: l, Filter: str})
		if err != nil {
			return sc, aphgrpc.HandleGetError(ctx, err)
		}
		if len(mc) == 0 {
			return sc, aphgrpc.HandleNotFoundError(ctx, fmt.Errorf("could not find any strains"))
		}
		var scdata []*stock.StockCollection_Data
		for _, m := range mc {
			scdata = append(scdata, &stock.StockCollection_Data{
				Type: s.GetResourceName(),
				Id:   m.Key,
				Attributes: &stock.StockAttributes{
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
					StrainProperties: &stock.StrainProperties{
						SystematicName: m.StrainProperties.SystematicName,
						Label:          m.StrainProperties.Label,
						Species:        m.StrainProperties.Species,
						Plasmid:        m.StrainProperties.Plasmid,
						Parent:         m.StrainProperties.Parent,
						Names:          m.StrainProperties.Names,
					},
				},
			})
		}
		if len(scdata) < int(r.Limit)-2 { // fewer results than limit
			sc.Data = scdata
			sc.Meta = &stock.Meta{Limit: r.Limit}
			return sc, nil
		}
		sc.Data = scdata[:len(scdata)-1]
		sc.Meta = &stock.Meta{
			Limit:      r.Limit,
			NextCursor: genNextCursorVal(scdata[len(scdata)-1]),
			Total:      int64(len(scdata)),
		}
	} else {
		mc, err := s.repo.ListStrains(&stock.StockParameters{Cursor: c, Limit: l})
		if err != nil {
			return sc, aphgrpc.HandleGetError(ctx, err)
		}
		if len(mc) == 0 {
			return sc, aphgrpc.HandleNotFoundError(ctx, fmt.Errorf("could not find any strains"))
		}
		var scdata []*stock.StockCollection_Data
		for _, m := range mc {
			scdata = append(scdata, &stock.StockCollection_Data{
				Type: s.GetResourceName(),
				Id:   m.Key,
				Attributes: &stock.StockAttributes{
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
					StrainProperties: &stock.StrainProperties{
						SystematicName: m.StrainProperties.SystematicName,
						Label:          m.StrainProperties.Label,
						Species:        m.StrainProperties.Species,
						Plasmid:        m.StrainProperties.Plasmid,
						Parent:         m.StrainProperties.Parent,
						Names:          m.StrainProperties.Names,
					},
				},
			})
		}
		if len(scdata) < int(r.Limit)-2 { // fewer results than limit
			sc.Data = scdata
			sc.Meta = &stock.Meta{Limit: r.Limit}
			return sc, nil
		}
		sc.Data = scdata[:len(scdata)-1]
		sc.Meta = &stock.Meta{
			Limit:      r.Limit,
			NextCursor: genNextCursorVal(scdata[len(scdata)-1]),
			Total:      int64(len(scdata)),
		}
	}
	return sc, nil
}

// ListPlasmids lists all existing plasmids
func (s *StockService) ListPlasmids(ctx context.Context, r *stock.StockParameters) (*stock.StockCollection, error) {
	sc := &stock.StockCollection{}
	c := r.Cursor
	l := r.Limit
	f := r.Filter
	if len(f) > 0 {
		p, err := query.ParseFilterString(f)
		if err != nil {
			return sc, fmt.Errorf("error parsing filter string: %s", err)
		}
		str, err := query.GenAQLFilterStatement(&query.StatementParameters{Fmap: arangodb.FMap, Filters: p, Doc: "s", Vert: "v"})
		if err != nil {
			return sc, fmt.Errorf("error generating AQL filter statement: %s", err)
		}
		// if the parsed statement is empty FILTER, just return empty string
		if str == "FILTER " {
			str = ""
		}
		mc, err := s.repo.ListPlasmids(&stock.StockParameters{Cursor: c, Limit: l, Filter: str})
		if err != nil {
			return sc, aphgrpc.HandleGetError(ctx, err)
		}
		if len(mc) == 0 {
			return sc, aphgrpc.HandleNotFoundError(ctx, fmt.Errorf("could not find any plasmids"))
		}
		var scdata []*stock.StockCollection_Data
		for _, m := range mc {
			scdata = append(scdata, &stock.StockCollection_Data{
				Type: s.GetResourceName(),
				Id:   m.Key,
				Attributes: &stock.StockAttributes{
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
					PlasmidProperties: &stock.PlasmidProperties{
						ImageMap: m.PlasmidProperties.ImageMap,
						Sequence: m.PlasmidProperties.Sequence,
					},
				},
			})
		}
		if len(scdata) < int(r.Limit)-2 { // fewer results than limit
			sc.Data = scdata
			sc.Meta = &stock.Meta{Limit: r.Limit}
			return sc, nil
		}
		sc.Data = scdata[:len(scdata)-1]
		sc.Meta = &stock.Meta{
			Limit:      r.Limit,
			NextCursor: genNextCursorVal(scdata[len(scdata)-1]),
			Total:      int64(len(scdata)),
		}
	} else {
		mc, err := s.repo.ListPlasmids(&stock.StockParameters{Cursor: c, Limit: l})
		if err != nil {
			return sc, aphgrpc.HandleGetError(ctx, err)
		}
		if len(mc) == 0 {
			return sc, aphgrpc.HandleNotFoundError(ctx, fmt.Errorf("could not find any plasmids"))
		}
		var scdata []*stock.StockCollection_Data
		for _, m := range mc {
			scdata = append(scdata, &stock.StockCollection_Data{
				Type: s.GetResourceName(),
				Id:   m.Key,
				Attributes: &stock.StockAttributes{
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
					PlasmidProperties: &stock.PlasmidProperties{
						ImageMap: m.PlasmidProperties.ImageMap,
						Sequence: m.PlasmidProperties.Sequence,
					},
				},
			})
		}
		if len(scdata) < int(r.Limit)-2 { // fewer results than limit
			sc.Data = scdata
			sc.Meta = &stock.Meta{Limit: r.Limit}
			return sc, nil
		}
		sc.Data = scdata[:len(scdata)-1]
		sc.Meta = &stock.Meta{
			Limit:      r.Limit,
			NextCursor: genNextCursorVal(scdata[len(scdata)-1]),
			Total:      int64(len(scdata)),
		}
	}
	return sc, nil
}

// RemoveStock removes an existing stock
func (s *StockService) RemoveStock(ctx context.Context, r *stock.StockId) (*empty.Empty, error) {
	e := &empty.Empty{}
	if err := r.Validate(); err != nil {
		return e, aphgrpc.HandleInvalidParamError(ctx, err)
	}
	if err := s.repo.RemoveStock(r.Id); err != nil {
		return e, aphgrpc.HandleDeleteError(ctx, err)
	}
	return e, nil
}

// LoadStock loads stocks with existing IDs into the database
func (s *StockService) LoadStock(ctx context.Context, r *stock.ExistingStock) (*stock.Stock, error) {
	st := &stock.Stock{}
	if err := r.Validate(); err != nil {
		return st, aphgrpc.HandleInvalidParamError(ctx, err)
	}
	id := r.Data.Id
	if id[:3] == "DBS" {
		m, err := s.repo.LoadStock(id, r)
		if err != nil {
			return st, aphgrpc.HandleInsertError(ctx, err)
		}
		st.Data = &stock.Stock_Data{
			Type: s.GetResourceName(),
			Id:   m.Key,
			Attributes: &stock.StockAttributes{
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
				StrainProperties: &stock.StrainProperties{
					SystematicName: m.StrainProperties.SystematicName,
					Label:          m.StrainProperties.Label,
					Species:        m.StrainProperties.Species,
					Plasmid:        m.StrainProperties.Plasmid,
					Parent:         m.StrainProperties.Parent,
					Names:          m.StrainProperties.Names,
				},
			},
		}
	} else {
		m, err := s.repo.LoadStock(id, r)
		if err != nil {
			return st, aphgrpc.HandleInsertError(ctx, err)
		}
		st.Data = &stock.Stock_Data{
			Type: s.GetResourceName(),
			Id:   m.Key,
			Attributes: &stock.StockAttributes{
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
				PlasmidProperties: &stock.PlasmidProperties{
					ImageMap: m.PlasmidProperties.ImageMap,
					Sequence: m.PlasmidProperties.Sequence,
				},
			},
		}
	}
	s.publisher.Publish(s.Topics["stockCreate"], st)
	return st, nil
}

func genNextCursorVal(scd *stock.StockCollection_Data) int64 {
	tint, _ := strconv.ParseInt(
		fmt.Sprintf("%d%d", scd.Attributes.CreatedAt.GetSeconds(), scd.Attributes.CreatedAt.GetNanos()),
		10,
		64,
	)
	return tint / 1000000
}
