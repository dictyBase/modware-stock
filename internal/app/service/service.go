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
	st.Data = &stock.Strain_Data{
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
			PlasmidProperties: &stock.PlasmidProperties{
				ImageMap: m.PlasmidProperties.ImageMap,
				Sequence: m.PlasmidProperties.Sequence,
			},
		},
	}
	return st, nil
}

// CreateStrain handles the creation of a new strain
func (s *StockService) CreateStrain(ctx context.Context, r *stock.NewStrain) (*stock.Strain, error) {
	st := &stock.Strain{}
	if err := r.Validate(); err != nil {
		return st, aphgrpc.HandleInvalidParamError(ctx, err)
	}
	m, err := s.repo.AddStrain(r)
	if err != nil {
		return st, aphgrpc.HandleInsertError(ctx, err)
	}
	st.Data = &stock.Strain_Data{
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
	s.publisher.PublishStrain(s.Topics["stockCreate"], st)
	return st, nil
}

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
			PlasmidProperties: &stock.PlasmidProperties{
				ImageMap: m.PlasmidProperties.ImageMap,
				Sequence: m.PlasmidProperties.Sequence,
			},
		},
	}
	s.publisher.PublishPlasmid(s.Topics["stockCreate"], st)
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
		return st, aphgrpc.HandleNotFoundError(ctx, fmt.Errorf("could not find strain with ID %s", m.ID))
	}
	st.Data = &stock.Strain_Data{
		Type: "strain",
		Id:   m.Key,
		Attributes: &stock.StrainAttributes{
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
	s.publisher.PublishStrain(s.Topics["stockUpdate"], st)
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
	if m.PlasmidProperties != nil {
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
				PlasmidProperties: &stock.PlasmidProperties{
					ImageMap: m.PlasmidProperties.ImageMap,
					Sequence: m.PlasmidProperties.Sequence,
				},
			},
		}
	} else {
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
			},
		}
	}
	s.publisher.PublishPlasmid(s.Topics["stockUpdate"], st)
	return st, nil
}

// ListStrains lists all existing strains
func (s *StockService) ListStrains(ctx context.Context, r *stock.StockParameters) (*stock.StrainCollection, error) {
	sc := &stock.StrainCollection{}
	var l int64
	c := r.Cursor
	f := r.Filter
	if r.Limit == 0 {
		l = 10
	} else {
		l = r.Limit
	}
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
		var scdata []*stock.StrainCollection_Data
		for _, m := range mc {
			scdata = append(scdata, &stock.StrainCollection_Data{
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
		if len(scdata) < int(l)-2 { // fewer results than limit
			sc.Data = scdata
			sc.Meta = &stock.Meta{Limit: l}
			return sc, nil
		}
		sc.Data = scdata[:len(scdata)-1]
		sc.Meta = &stock.Meta{
			Limit:      l,
			NextCursor: genNextStrainCursorVal(scdata[len(scdata)-1]),
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
		var scdata []*stock.StrainCollection_Data
		for _, m := range mc {
			scdata = append(scdata, &stock.StrainCollection_Data{
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
		if len(scdata) < int(l)-2 { // fewer results than limit
			sc.Data = scdata
			sc.Meta = &stock.Meta{Limit: l}
			return sc, nil
		}
		sc.Data = scdata[:len(scdata)-1]
		sc.Meta = &stock.Meta{
			Limit:      l,
			NextCursor: genNextStrainCursorVal(scdata[len(scdata)-1]),
			Total:      int64(len(scdata)),
		}
	}
	return sc, nil
}

// ListPlasmids lists all existing plasmids
func (s *StockService) ListPlasmids(ctx context.Context, r *stock.StockParameters) (*stock.PlasmidCollection, error) {
	sc := &stock.PlasmidCollection{}
	var l int64
	c := r.Cursor
	f := r.Filter
	if r.Limit == 0 {
		l = 10
	} else {
		l = r.Limit
	}
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
		var scdata []*stock.PlasmidCollection_Data
		for _, m := range mc {
			scdata = append(scdata, &stock.PlasmidCollection_Data{
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
					PlasmidProperties: &stock.PlasmidProperties{
						ImageMap: m.PlasmidProperties.ImageMap,
						Sequence: m.PlasmidProperties.Sequence,
					},
				},
			})
		}
		if len(scdata) < int(l)-2 { // fewer results than limit
			sc.Data = scdata
			sc.Meta = &stock.Meta{Limit: l}
			return sc, nil
		}
		sc.Data = scdata[:len(scdata)-1]
		sc.Meta = &stock.Meta{
			Limit:      l,
			NextCursor: genNextPlasmidCursorVal(scdata[len(scdata)-1]),
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
		var scdata []*stock.PlasmidCollection_Data
		for _, m := range mc {
			scdata = append(scdata, &stock.PlasmidCollection_Data{
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
					PlasmidProperties: &stock.PlasmidProperties{
						ImageMap: m.PlasmidProperties.ImageMap,
						Sequence: m.PlasmidProperties.Sequence,
					},
				},
			})
		}
		if len(scdata) < int(l)-2 { // fewer results than limit
			sc.Data = scdata
			sc.Meta = &stock.Meta{Limit: l}
			return sc, nil
		}
		sc.Data = scdata[:len(scdata)-1]
		sc.Meta = &stock.Meta{
			Limit:      l,
			NextCursor: genNextPlasmidCursorVal(scdata[len(scdata)-1]),
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
			Type: "strain",
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
			Type: "plasmid",
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
	s.publisher.PublishStock(s.Topics["stockCreate"], st)
	return st, nil
}

func genNextStrainCursorVal(scd *stock.StrainCollection_Data) int64 {
	tint, _ := strconv.ParseInt(
		fmt.Sprintf("%d%d", scd.Attributes.CreatedAt.GetSeconds(), scd.Attributes.CreatedAt.GetNanos()),
		10,
		64,
	)
	return tint / 1000000
}

func genNextPlasmidCursorVal(pcd *stock.PlasmidCollection_Data) int64 {
	tint, _ := strconv.ParseInt(
		fmt.Sprintf("%d%d", pcd.Attributes.CreatedAt.GetSeconds(), pcd.Attributes.CreatedAt.GetNanos()),
		10,
		64,
	)
	return tint / 1000000
}
