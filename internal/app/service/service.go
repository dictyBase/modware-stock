package service

import (
	"context"
	"fmt"
	"strconv"

	"github.com/dictyBase/apihelpers/aphgrpc"
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
	m, err := s.repo.GetStock(r.Id)
	if err != nil {
		return st, aphgrpc.HandleGetError(ctx, err)
	}
	if m.NotFound {
		return st, aphgrpc.HandleNotFoundError(ctx, err)
	}
	st.Data = &stock.Stock_Data{
		Type: s.GetResourceName(),
		Id:   m.Key, // need to make sure this is DBS/DBP ID
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
				SystematicName: m.SystematicName,
				Descriptor_:    m.Descriptor,
				Species:        m.Species,
				Plasmid:        m.Plasmid,
				Parents:        m.Parents,
				Names:          m.Names,
			},
			PlasmidProperties: &stock.PlasmidProperties{
				ImageMap: m.ImageMap,
				Sequence: m.Sequence,
			},
		},
	}
	return st, nil
}

// CreateStock handles the creation of a new stock
func (s *StockService) CreateStock(ctx context.Context, r *stock.NewStock) (*stock.Stock, error) {
	st := &stock.Stock{}
	if err := r.Validate(); err != nil {
		return st, aphgrpc.HandleInvalidParamError(ctx, err)
	}
	// m, err := s.repo.AddStock(r)
	// if err != nil {
	// 	return st, aphgrpc.HandleInsertError(ctx, err)
	// }
	// if m.NotFound {
	// 	return st, aphgrpc.HandleNotFoundError(ctx, err)
	// }
	// st.Data = &stock.Stock_Data{
	// 	Type: s.GetResourceName(),
	// 	Id:   m.Key, // need to make sure this is DBS/DBP ID
	// 	Attributes: &stock.StockAttributes{
	// 		CreatedAt:       aphgrpc.TimestampProto(m.CreatedAt),
	// 		UpdatedAt:       aphgrpc.TimestampProto(m.UpdatedAt),
	// 		CreatedBy:       m.CreatedBy,
	// 		UpdatedBy:       m.UpdatedBy,
	// 		Summary:         m.Summary,
	// 		EditableSummary: m.EditableSummary,
	// 		Depositor:       m.Depositor,
	// 		Genes:           m.Genes,
	// 		Dbxrefs:         m.Dbxrefs,
	// 		Publications:    m.Publications,
	// 		StrainProperties: &stock.StrainProperties{
	// 			SystematicName: m.SystematicName,
	// 			Descriptor_:    m.Descriptor,
	// 			Species:        m.Species,
	// 			Plasmid:        m.Plasmid,
	// 			Parents:        m.Parents,
	// 			Names:          m.Names,
	// 		},
	// 		PlasmidProperties: &stock.PlasmidProperties{
	// 			ImageMap: m.ImageMap,
	// 			Sequence: m.Sequence,
	// 		},
	// 	},
	// }
	s.publisher.Publish(s.Topics["stockCreate"], st)
	return st, nil
}

// UpdateStock handles updating an existing stock
func (s *StockService) UpdateStock(ctx context.Context, r *stock.StockUpdate) (*stock.Stock, error) {
	st := &stock.Stock{}
	if err := r.Validate(); err != nil {
		return st, aphgrpc.HandleInvalidParamError(ctx, err)
	}
	m, err := s.repo.EditStock(r)
	if err != nil {
		return st, aphgrpc.HandleUpdateError(ctx, err)
	}
	if m.NotFound {
		return st, aphgrpc.HandleNotFoundError(ctx, err)
	}
	st.Data = &stock.Stock_Data{
		Type: s.GetResourceName(),
		Id:   m.Key, // need to make sure this is DBS/DBP ID
		Attributes: &stock.StockAttributes{
			UpdatedBy:       m.UpdatedBy,
			Summary:         m.Summary,
			EditableSummary: m.EditableSummary,
			Depositor:       m.Depositor,
			Genes:           m.Genes,
			Dbxrefs:         m.Dbxrefs,
			Publications:    m.Publications,
			StrainProperties: &stock.StrainProperties{
				SystematicName: m.SystematicName,
				Descriptor_:    m.Descriptor,
				Species:        m.Species,
				Plasmid:        m.Plasmid,
				Parents:        m.Parents,
				Names:          m.Names,
			},
			PlasmidProperties: &stock.PlasmidProperties{
				ImageMap: m.ImageMap,
				Sequence: m.Sequence,
			},
		},
	}
	s.publisher.Publish(s.Topics["stockUpdate"], st)
	return st, nil
}

// ListStocks lists all existing stocks
func (s *StockService) ListStocks(ctx context.Context, r *stock.ListParameters) (*stock.StockCollection, error) {
	sc := &stock.StockCollection{}
	if len(r.Filter) == 0 { // no filter parameters
		mc, err := s.repo.ListStocks(r.Cursor, r.Limit)
		if err != nil {
			return sc, aphgrpc.HandleGetError(ctx, err)
		}
		if len(mc) == 0 {
			return sc, aphgrpc.HandleNotFoundError(ctx, err)
		}
		var scdata []*stock.StockCollection_Data
		for _, m := range mc {
			scdata = append(scdata, &stock.StockCollection_Data{
				Type: s.GetResourceName(),
				Id:   m.Key, // need to make sure this is DBS/DBP ID
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
						SystematicName: m.SystematicName,
						Descriptor_:    m.Descriptor,
						Species:        m.Species,
						Plasmid:        m.Plasmid,
						Parents:        m.Parents,
						Names:          m.Names,
					},
					PlasmidProperties: &stock.PlasmidProperties{
						ImageMap: m.ImageMap,
						Sequence: m.Sequence,
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
		}
	}
	return sc, nil
}

// ListStrains lists all existing strains
func (s *StockService) ListStrains(ctx context.Context, r *stock.StrainListParameters) (*stock.StockCollection, error) {
	sc := &stock.StockCollection{}
	if len(r.Filter) == 0 { // no filter parameters
		mc, err := s.repo.ListStocks(r.Cursor, r.Limit)
		if err != nil {
			return sc, aphgrpc.HandleGetError(ctx, err)
		}
		if len(mc) == 0 {
			return sc, aphgrpc.HandleNotFoundError(ctx, err)
		}
		var scdata []*stock.StockCollection_Data
		for _, m := range mc {
			scdata = append(scdata, &stock.StockCollection_Data{
				Type: s.GetResourceName(),
				Id:   m.Key, // need to make sure this is DBS/DBP ID
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
						SystematicName: m.SystematicName,
						Descriptor_:    m.Descriptor,
						Species:        m.Species,
						Plasmid:        m.Plasmid,
						Parents:        m.Parents,
						Names:          m.Names,
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
		}
	}
	return sc, nil
}

// ListPlasmids lists all existing plasmids
func (s *StockService) ListPlasmids(ctx context.Context, r *stock.PlasmidListParameters) (*stock.StockCollection, error) {
	sc := &stock.StockCollection{}
	if len(r.Filter) == 0 { // no filter parameters
		mc, err := s.repo.ListStocks(r.Cursor, r.Limit)
		if err != nil {
			return sc, aphgrpc.HandleGetError(ctx, err)
		}
		if len(mc) == 0 {
			return sc, aphgrpc.HandleNotFoundError(ctx, err)
		}
		var scdata []*stock.StockCollection_Data
		for _, m := range mc {
			scdata = append(scdata, &stock.StockCollection_Data{
				Type: s.GetResourceName(),
				Id:   m.Key, // need to make sure this is DBS/DBP ID
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
						ImageMap: m.ImageMap,
						Sequence: m.Sequence,
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

func genNextCursorVal(scd *stock.StockCollection_Data) int64 {
	tint, _ := strconv.ParseInt(
		fmt.Sprintf("%d%d", scd.Attributes.CreatedAt.GetSeconds(), scd.Attributes.CreatedAt.GetNanos()),
		10,
		64,
	)
	return tint / 1000000
}
