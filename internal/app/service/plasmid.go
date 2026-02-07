package service

import (
	"context"

	F "github.com/IBM/fp-go/function"
	IOE "github.com/IBM/fp-go/ioeither"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
)

// CreatePlasmid handles the creation of a new plasmid using IOEither composition
func (s *StockService) CreatePlasmid(
	ctx context.Context,
	req *stock.NewPlasmid,
) (*stock.Plasmid, error) {
	result := F.Pipe6(
		IOE.Of[error](createPlasmidContext{
			ctx:       ctx,
			request:   req,
			repo:      s.repo,
			params:    s.Params,
			topics:    s.Topics,
			publisher: s.publisher,
		}),
		IOE.Bind(setValidatedNewPlasmid, validateNewPlasmidRequest),
		IOE.Bind(setCreatedPlasmidDoc, createPlasmidInRepository),
		IOE.Let[error](setCreatedPlasmidData, transformToCreatedPlasmidData),
		IOE.Chain(publishCreatedPlasmid),
		IOE.Map[error](extractCreatePlasmidResponse),
		toCreatePlasmidResult(ctx),
	)
	return result.F1, result.F2
}

// GetPlasmid handles getting a plasmid by its ID using IOEither composition
func (s *StockService) GetPlasmid(
	ctx context.Context,
	req *stock.StockId,
) (*stock.Plasmid, error) {
	result := F.Pipe6(
		IOE.Of[error](getPlasmidContext{
			ctx:     ctx,
			request: req,
			repo:    s.repo,
		}),
		IOE.Bind(setValidatedRequest, validatePlasmidRequest),
		IOE.Bind(setStockDocument, retrievePlasmidFromRepository),
		IOE.Map[error](getStockDoc),
		IOE.Map[error](makePlasmidData),
		IOE.Map[error](extractPlasmidResponse),
		toServiceResult(ctx),
	)
	return result.F1, result.F2
}

// LoadPlasmid loads plasmids with existing IDs into the database using IOEither composition
func (s *StockService) LoadPlasmid(
	ctx context.Context,
	req *stock.ExistingPlasmid,
) (*stock.Plasmid, error) {
	result := F.Pipe6(
		IOE.Of[error](loadPlasmidContext{
			ctx:       ctx,
			request:   req,
			repo:      s.repo,
			params:    s.Params,
			topics:    s.Topics,
			publisher: s.publisher,
		}),
		IOE.Bind(setValidatedExistingPlasmid, validateExistingPlasmidRequest),
		IOE.Bind(setLoadedPlasmidDoc, loadPlasmidInRepository),
		IOE.Let[error](setLoadedPlasmidData, transformToLoadedPlasmidData),
		IOE.Chain(publishLoadedPlasmid),
		IOE.Map[error](extractLoadPlasmidResponse),
		toLoadPlasmidResult(ctx),
	)
	return result.F1, result.F2
}

// UpdatePlasmid handles updating an existing plasmid using IOEither composition
func (s *StockService) UpdatePlasmid(
	ctx context.Context,
	req *stock.PlasmidUpdate,
) (*stock.Plasmid, error) {
	result := F.Pipe7(
		IOE.Of[error](updatePlasmidContext{
			ctx:       ctx,
			request:   req,
			repo:      s.repo,
			topics:    s.Topics,
			publisher: s.publisher,
		}),
		IOE.Bind(setValidatedUpdateRequest, validateUpdatePlasmidRequest),
		IOE.Bind(setUpdatedPlasmidDoc, updatePlasmidInRepository),
		IOE.Bind(setFullPlasmidDoc, retrieveFullPlasmidDoc),
		IOE.Let[error](setUpdatedPlasmidData, transformToUpdatedPlasmidData),
		IOE.Chain(publishUpdatedPlasmid),
		IOE.Map[error](extractUpdatePlasmidResponse),
		toUpdatePlasmidResult(ctx),
	)
	return result.F1, result.F2
}

// ListPlasmids lists all existing plasmids using IOEither composition
func (s *StockService) ListPlasmids(
	ctx context.Context,
	param *stock.StockParameters,
) (*stock.PlasmidCollection, error) {
	limit := limitVal(param.Limit)

	result := F.Pipe6(
		IOE.Of[error](listPlasmidsContext{
			ctx:   ctx,
			param: param,
			limit: limit,
			repo:  s.repo,
		}),
		IOE.Bind(setValidatedFilter, validatePlasmidFilter),
		IOE.Bind(setStockDocList, retrievePlasmidsFromRepository),
		IOE.Let[error](
			setPlasmidCollectionData,
			transformToPlasmidCollection,
		),
		IOE.Let[error](setNextCursor, computeNextCursor),
		IOE.Map[error](extractPlasmidCollectionResponse),
		toPlasmidCollectionResult(ctx, limit),
	)
	return result.F1, result.F2
}
