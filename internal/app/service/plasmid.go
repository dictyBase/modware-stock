package service

import (
	"context"
	"fmt"

	F "github.com/IBM/fp-go/function"
	IOE "github.com/IBM/fp-go/ioeither"
	"github.com/dictyBase/aphgrpc"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
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

// GetPlasmid handles getting a plasmid by its ID using IOEither composition
func (s *StockService) GetPlasmid(
	ctx context.Context,
	req *stock.StockId,
) (*stock.Plasmid, error) {
	result := F.Pipe5(
		IOE.Of[error](getPlasmidContext{
			ctx:     ctx,
			request: req,
			repo:    s.repo,
		}),
		IOE.Bind(setValidatedRequest, validatePlasmidRequest),
		IOE.Bind(setStockDocument, retrievePlasmidFromRepository),
		IOE.Let[error](setPlasmidData, transformToPlasmidData),
		IOE.Map[error](extractPlasmidResponse),
		toServiceResult(ctx),
	)
	return result.F1, result.F2
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
	// Fetch the complete plasmid record to get all fields including ontology
	fullPlasmid, err := toTuple(s.repo.GetPlasmid(r.Data.Id))
	if err != nil {
		return plasmid, aphgrpc.HandleGetError(ctx, err)
	}
	plasmid.Data = makePlasmidData(fullPlasmid)
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
	plasmidCollection := &stock.PlasmidCollection{
		Meta: &stock.Meta{Limit: limit},
	}
	stockDocs, err := stockModelList(&modelListParams{
		ctx:         ctx,
		stockParams: param,
		limit:       limit,
		fn:          s.repo.ListPlasmids,
	})
	if err != nil {
		return plasmidCollection, err
	}
	pdata := plasmidModelToCollectionSlice(stockDocs)
	if len(pdata) < int(limit)-2 { // fewer results than limit
		plasmidCollection.Data = pdata
		plasmidCollection.Meta.Total = int64(len(pdata))
		return plasmidCollection, nil
	}
	plasmidCollection.Data = pdata[:len(pdata)-1]
	plasmidCollection.Meta.NextCursor = genNextCursorVal(
		pdata[len(pdata)-1].Attributes.CreatedAt,
	)
	plasmidCollection.Meta.Total = int64(len(pdata))
	return plasmidCollection, nil
}
