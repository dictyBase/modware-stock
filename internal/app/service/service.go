package service

import (
	"context"
	"fmt"
	"time"

	"github.com/dictyBase/apihelpers/aphgrpc"
	"github.com/dictyBase/arangomanager/query"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/message"
	"github.com/dictyBase/modware-stock/internal/repository"
	"github.com/dictyBase/modware-stock/internal/repository/arangodb"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/golang/protobuf/ptypes/timestamp"
)

var stockProp = map[string]int{
	"label":        1,
	"species":      1,
	"plasmid":      1,
	"parent":       1,
	"name":         1,
	"image_map":    1,
	"sequence":     1,
	"plasmid_name": 1,
}

// StockService is the container for managing stock service
// definition
type StockService struct {
	*aphgrpc.Service
	repo      repository.StockRepository
	publisher message.Publisher
	stock.UnimplementedStockServiceServer
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
			Label:           m.StrainProperties.Label,
			Species:         m.StrainProperties.Species,
			Plasmid:         m.StrainProperties.Plasmid,
			Parent:          m.StrainProperties.Parent,
			Names:           m.StrainProperties.Names,
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
			ImageMap:        m.PlasmidProperties.ImageMap,
			Sequence:        m.PlasmidProperties.Sequence,
			Name:            m.PlasmidProperties.Name,
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
			Label:           m.StrainProperties.Label,
			Species:         m.StrainProperties.Species,
			Plasmid:         m.StrainProperties.Plasmid,
			Parent:          m.StrainProperties.Parent,
			Names:           m.StrainProperties.Names,
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
			ImageMap:        m.PlasmidProperties.ImageMap,
			Sequence:        m.PlasmidProperties.Sequence,
			Name:            m.PlasmidProperties.Name,
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
			Label:           m.StrainProperties.Label,
			Species:         m.StrainProperties.Species,
			Plasmid:         m.StrainProperties.Plasmid,
			Parent:          m.StrainProperties.Parent,
			Names:           m.StrainProperties.Names,
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
		return sl, aphgrpc.HandleNotFoundError(ctx, fmt.Errorf("could not find any strains"))
	}
	sdata := make([]*stock.StrainList_Data, 0)
	for _, m := range mc {
		sdata = append(sdata, &stock.StrainList_Data{
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
	sl.Data = sdata
	return sl, nil
}

// ListStrains lists all existing strains
func (s *StockService) ListStrains(ctx context.Context, r *stock.StockParameters) (*stock.StrainCollection, error) {
	sc := &stock.StrainCollection{}
	limit := int64(10)
	if r.Limit > 10 {
		limit = r.Limit
	}
	var astmt string
	var vert bool
	if len(r.Filter) > 0 {
		p, err := query.ParseFilterString(r.Filter)
		if err != nil {
			return sc, aphgrpc.HandleInvalidParamError(
				ctx,
				fmt.Errorf("error in parsing filter string"),
			)
		}
		// need to check if filter contains an item found in strain properties
		for _, n := range p {
			if _, ok := stockProp[n.Field]; ok {
				vert = true
				break
			}
		}
		if vert {
			astmt, err = query.GenAQLFilterStatement(&query.StatementParameters{Fmap: arangodb.FMap, Filters: p, Vert: "v"})
			if err != nil {
				return sc, aphgrpc.HandleInvalidParamError(
					ctx,
					fmt.Errorf("error in generating AQL statement"),
				)
			}
		} else {
			astmt, err = query.GenAQLFilterStatement(&query.StatementParameters{Fmap: arangodb.FMap, Filters: p, Doc: "s"})
			if err != nil {
				return sc, aphgrpc.HandleInvalidParamError(
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
	mc, err := s.repo.ListStrains(&stock.StockParameters{Cursor: r.Cursor, Limit: limit, Filter: astmt})
	if err != nil {
		return sc, aphgrpc.HandleGetError(ctx, err)
	}
	if len(mc) == 0 {
		return sc, aphgrpc.HandleNotFoundError(ctx, fmt.Errorf("could not find any strains"))
	}
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
	if len(sdata) < int(limit)-2 { // fewer results than limit
		sc.Data = sdata
		sc.Meta = &stock.Meta{
			Limit: limit,
			Total: int64(len(sdata)),
		}
		return sc, nil
	}
	sc.Data = sdata[:len(sdata)-1]
	sc.Meta = &stock.Meta{
		Limit:      limit,
		NextCursor: genNextCursorVal(sdata[len(sdata)-1].Attributes.CreatedAt),
		Total:      int64(len(sdata)),
	}
	return sc, nil
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

// LoadStock loads strains with existing IDs into the database
func (s *StockService) LoadStrain(ctx context.Context, r *stock.ExistingStrain) (*stock.Strain, error) {
	st := &stock.Strain{}
	if err := r.Validate(); err != nil {
		return st, aphgrpc.HandleInvalidParamError(ctx, err)
	}
	id := r.Data.Id
	m, err := s.repo.LoadStrain(id, r)
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
			Label:           m.StrainProperties.Label,
			Species:         m.StrainProperties.Species,
			Plasmid:         m.StrainProperties.Plasmid,
			Parent:          m.StrainProperties.Parent,
			Names:           m.StrainProperties.Names,
		},
	}
	s.publisher.PublishStrain(s.Topics["stockCreate"], st)
	return st, nil
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

func genNextCursorVal(c *timestamp.Timestamp) int64 {
	ts := ptypes.TimestampString(c)
	t, _ := time.Parse("2006-01-02T15:04:05Z", ts)
	return t.UnixNano() / 1000000
}
