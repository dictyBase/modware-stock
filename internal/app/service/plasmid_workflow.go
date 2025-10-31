package service

import (
	"context"
	"fmt"

	E "github.com/IBM/fp-go/either"
	fperrors "github.com/IBM/fp-go/errors"
	F "github.com/IBM/fp-go/function"
	IOE "github.com/IBM/fp-go/ioeither"
	T "github.com/IBM/fp-go/tuple"
	"github.com/dictyBase/aphgrpc"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/model"
)

// Curried setters for building context
var (
	// setValidatedRequest sets validated request ID in context
	setValidatedRequest = F.Curry2(
		func(validID string, pctx getPlasmidContext) withValidatedRequest {
			return withValidatedRequest{
				getPlasmidContext: pctx,
				validatedID:       validID,
			}
		},
	)

	// setStockDocument sets stock document in context
	setStockDocument = F.Curry2(
		func(doc *model.StockDoc, pctx withValidatedRequest) withStockDocument {
			return withStockDocument{
				withValidatedRequest: pctx,
				stockDoc:             doc,
			}
		},
	)

	// setPlasmidData sets plasmid data in context
	setPlasmidData = F.Curry2(
		func(data *stock.Plasmid_Data, pctx withStockDocument) withPlasmidData {
			return withPlasmidData{
				withStockDocument: pctx,
				plasmidData:       data,
			}
		},
	)

	// extractPlasmidResponse extracts plasmid response from enriched context
	extractPlasmidResponse = func(pctx withPlasmidData) *stock.Plasmid {
		return &stock.Plasmid{Data: pctx.plasmidData}
	}
)

// validatePlasmidRequest validates the stock ID request
func validatePlasmidRequest(
	pctx getPlasmidContext,
) IOE.IOEither[error, string] {
	return func() E.Either[error, string] {
		if err := pctx.request.Validate(); err != nil {
			return E.Left[string](
				fmt.Errorf("invalid request parameters: %w", err),
			)
		}
		return E.Right[error](pctx.request.Id)
	}
}

// retrievePlasmidFromRepository retrieves plasmid from repository
func retrievePlasmidFromRepository(
	pctx withValidatedRequest,
) IOE.IOEither[error, *model.StockDoc] {
	return F.Pipe1(
		pctx.repo.GetPlasmid(pctx.validatedID),
		IOE.MapLeft[*model.StockDoc](
			fperrors.OnError(
				fmt.Sprintf("failed to retrieve plasmid %s", pctx.validatedID),
			),
		),
	)
}

// transformToPlasmidData transforms stock document to plasmid data using point-free composition
var transformToPlasmidData = F.Flow2(
	func(pctx withStockDocument) *model.StockDoc { return pctx.stockDoc },
	makePlasmidData,
)

// toServiceResult converts IOEither result to service response tuple with error handling
func toServiceResult(ctx context.Context) PlasmidConverter {
	return func(ioe PlasmidIO) PlasmidResult {
		return F.Pipe1(
			ioe(),
			E.Fold(
				func(e error) PlasmidResult {
					return T.MakeTuple2(
						&stock.Plasmid{},
						aphgrpc.HandleGetError(ctx, e),
					)
				},
				func(p *stock.Plasmid) PlasmidResult {
					return T.MakeTuple2[*stock.Plasmid, error](
						p,
						nil,
					)
				},
			),
		)
	}
}

// ListPlasmids workflow functions

// Curried setters for ListPlasmids context building
var (
	// setValidatedFilter sets validated filter in context
	setValidatedFilter = F.Curry2(
		func(filter string, lctx listPlasmidsContext) withValidatedFilter {
			return withValidatedFilter{
				listPlasmidsContext: lctx,
				validatedFilter:     filter,
			}
		},
	)

	// setStockDocList sets stock document list in context
	setStockDocList = F.Curry2(
		func(docs []*model.StockDoc, lctx withValidatedFilter) withStockDocList {
			return withStockDocList{
				withValidatedFilter: lctx,
				stockDocs:           docs,
			}
		},
	)

	// setPlasmidCollectionData sets plasmid collection data in context
	setPlasmidCollectionData = F.Curry2(
		func(data []*stock.PlasmidCollection_Data, lctx withStockDocList) withPlasmidCollectionData {
			return withPlasmidCollectionData{
				withStockDocList: lctx,
				collectionData:   data,
			}
		},
	)

	// setNextCursor sets next cursor in context
	setNextCursor = F.Curry2(
		func(cursor int64, lctx withPlasmidCollectionData) withNextCursor {
			return withNextCursor{
				withPlasmidCollectionData: lctx,
				nextCursor:                cursor,
			}
		},
	)

	// extractPlasmidCollectionResponse extracts collection response from enriched context
	extractPlasmidCollectionResponse = func(lctx withNextCursor) *stock.PlasmidCollection {
		pdata := lctx.collectionData

		// If we have a next cursor, slice the data to exclude the last item
		if lctx.nextCursor > 0 && len(pdata) > 0 {
			pdata = pdata[:len(pdata)-1]
		}

		return &stock.PlasmidCollection{
			Data: pdata,
			Meta: &stock.Meta{
				Limit:      lctx.limit,
				Total:      int64(len(pdata)),
				NextCursor: lctx.nextCursor,
			},
		}
	}
)

// validatePlasmidFilter validates and processes the filter parameter
func validatePlasmidFilter(
	lctx listPlasmidsContext,
) IOE.IOEither[error, string] {
	return func() E.Either[error, string] {
		astmt, err := stockAQLStatement(lctx.param.Filter)
		if err != nil {
			return E.Left[string](
				fmt.Errorf("invalid filter parameter: %w", err),
			)
		}
		return E.Right[error](astmt)
	}
}

// retrievePlasmidsFromRepository retrieves plasmids from repository
func retrievePlasmidsFromRepository(
	lctx withValidatedFilter,
) IOE.IOEither[error, []*model.StockDoc] {
	return F.Pipe1(
		lctx.repo.ListPlasmids(&stock.StockParameters{
			Cursor: lctx.param.Cursor,
			Limit:  lctx.limit,
			Filter: lctx.validatedFilter,
		}),
		IOE.MapLeft[[]*model.StockDoc](
			fperrors.OnError("failed to retrieve plasmids from repository"),
		),
	)
}

// transformToPlasmidCollection transforms stock documents to plasmid collection data
var transformToPlasmidCollection = F.Flow2(
	func(lctx withStockDocList) []*model.StockDoc { return lctx.stockDocs },
	plasmidModelToCollectionSlice,
)

// computeNextCursor computes the next cursor value based on results
var computeNextCursor = func(lctx withPlasmidCollectionData) int64 {
	pdata := lctx.collectionData
	if len(pdata) < int(lctx.limit)-2 {
		return 0 // No next cursor for incomplete result sets
	}
	// Return cursor value from last item
	if len(pdata) > 0 {
		return genNextCursorVal(pdata[len(pdata)-1].Attributes.CreatedAt)
	}
	return 0
}

// toPlasmidCollectionResult converts IOEither result to collection response tuple
func toPlasmidCollectionResult(
	ctx context.Context,
	limit int64,
) PlasmidCollectionConverter {
	return func(ioe PlasmidCollectionIO) PlasmidCollectionResult {
		return F.Pipe1(
			ioe(),
			E.Fold(
				func(e error) PlasmidCollectionResult {
					return T.MakeTuple2(
						&stock.PlasmidCollection{
							Meta: &stock.Meta{Limit: limit},
						},
						aphgrpc.HandleGetError(ctx, e),
					)
				},
				func(collection *stock.PlasmidCollection) PlasmidCollectionResult {
					return T.MakeTuple2[*stock.PlasmidCollection, error](
						collection,
						nil,
					)
				},
			),
		)
	}
}

// CreatePlasmid workflow functions

// Curried setters for CreatePlasmid context building
var (
	// setValidatedNewPlasmid sets validated new plasmid request in context
	setValidatedNewPlasmid = F.Curry2(
		func(req *stock.NewPlasmid, cctx createPlasmidContext) withValidatedNewPlasmid {
			return withValidatedNewPlasmid{
				createPlasmidContext: cctx,
				validatedRequest:     req,
			}
		},
	)

	// setCreatedPlasmidDoc sets created stock document in context
	setCreatedPlasmidDoc = F.Curry2(
		func(doc *model.StockDoc, cctx withValidatedNewPlasmid) withCreatedPlasmidDoc {
			return withCreatedPlasmidDoc{
				withValidatedNewPlasmid: cctx,
				stockDoc:                doc,
			}
		},
	)

	// setCreatedPlasmidData sets created plasmid data in context
	setCreatedPlasmidData = F.Curry2(
		func(data *stock.Plasmid_Data, cctx withCreatedPlasmidDoc) withCreatedPlasmidData {
			return withCreatedPlasmidData{
				withCreatedPlasmidDoc: cctx,
				plasmidData:           data,
			}
		},
	)

	// extractCreatePlasmidResponse extracts plasmid response from enriched context
	extractCreatePlasmidResponse = func(cctx withCreatedPlasmidData) *stock.Plasmid {
		return &stock.Plasmid{Data: cctx.plasmidData}
	}
)

// validateNewPlasmidRequest validates the new plasmid request
func validateNewPlasmidRequest(
	cctx createPlasmidContext,
) IOE.IOEither[error, *stock.NewPlasmid] {
	return func() E.Either[error, *stock.NewPlasmid] {
		if err := cctx.request.Validate(); err != nil {
			return E.Left[*stock.NewPlasmid](
				fmt.Errorf("invalid request parameters: %w", err),
			)
		}

		// Apply default plasmid term if not provided
		if len(cctx.request.Data.Attributes.DictyPlasmidProperty) == 0 {
			cctx.request.Data.Attributes.DictyPlasmidProperty = cctx.params["plasmid_term"]
		}

		return E.Right[error](cctx.request)
	}
}

// createPlasmidInRepository creates plasmid in repository
func createPlasmidInRepository(
	cctx withValidatedNewPlasmid,
) IOE.IOEither[error, *model.StockDoc] {
	return F.Pipe1(
		cctx.repo.AddPlasmid(cctx.validatedRequest),
		IOE.MapLeft[*model.StockDoc](
			fperrors.OnError("failed to create plasmid in repository"),
		),
	)
}

// transformToCreatedPlasmidData transforms stock document to plasmid data
var transformToCreatedPlasmidData = F.Flow2(
	func(cctx withCreatedPlasmidDoc) *model.StockDoc { return cctx.stockDoc },
	makePlasmidData,
)

// publishCreatedPlasmid publishes the created plasmid event
func publishCreatedPlasmid(
	cctx withCreatedPlasmidData,
) IOE.IOEither[error, withCreatedPlasmidData] {
	return IOE.TryCatchError(
		func() (withCreatedPlasmidData, error) {
			plasmid := &stock.Plasmid{Data: cctx.plasmidData}
			err := cctx.publisher.PublishPlasmid(
				cctx.topics["stockCreate"],
				plasmid,
			)
			if err != nil {
				return cctx, fmt.Errorf("failed to publish plasmid creation event: %w", err)
			}
			return cctx, nil
		},
	)
}

// toCreatePlasmidResult converts IOEither result to service response tuple
func toCreatePlasmidResult(ctx context.Context) PlasmidConverter {
	return func(ioe PlasmidIO) PlasmidResult {
		return F.Pipe1(
			ioe(),
			E.Fold(
				func(e error) PlasmidResult {
					return T.MakeTuple2(
						&stock.Plasmid{},
						aphgrpc.HandleInsertError(ctx, e),
					)
				},
				func(plasmid *stock.Plasmid) PlasmidResult {
					return T.MakeTuple2[*stock.Plasmid, error](
						plasmid,
						nil,
					)
				},
			),
		)
	}
}

// UpdatePlasmid workflow functions

// Curried setters for UpdatePlasmid context building
var (
	// setValidatedUpdateRequest sets validated update request in context
	setValidatedUpdateRequest = F.Curry2(
		func(req *stock.PlasmidUpdate, uctx updatePlasmidContext) withValidatedUpdate {
			return withValidatedUpdate{
				updatePlasmidContext: uctx,
				validatedRequest:     req,
			}
		},
	)

	// setUpdatedPlasmidDoc sets updated stock document in context
	setUpdatedPlasmidDoc = F.Curry2(
		func(doc *model.StockDoc, uctx withValidatedUpdate) withUpdatedPlasmidDoc {
			return withUpdatedPlasmidDoc{
				withValidatedUpdate: uctx,
				stockDoc:            doc,
			}
		},
	)

	// setFullPlasmidDoc sets full plasmid document in context
	setFullPlasmidDoc = F.Curry2(
		func(doc *model.StockDoc, uctx withUpdatedPlasmidDoc) withFullPlasmidDoc {
			return withFullPlasmidDoc{
				withUpdatedPlasmidDoc: uctx,
				fullStockDoc:          doc,
			}
		},
	)

	// setUpdatedPlasmidData sets updated plasmid data in context
	setUpdatedPlasmidData = F.Curry2(
		func(data *stock.Plasmid_Data, uctx withFullPlasmidDoc) withUpdatedPlasmidData {
			return withUpdatedPlasmidData{
				withFullPlasmidDoc: uctx,
				plasmidData:        data,
			}
		},
	)

	// extractUpdatePlasmidResponse extracts plasmid response from enriched context
	extractUpdatePlasmidResponse = func(uctx withUpdatedPlasmidData) *stock.Plasmid {
		return &stock.Plasmid{Data: uctx.plasmidData}
	}
)

// validateUpdatePlasmidRequest validates the update plasmid request
func validateUpdatePlasmidRequest(
	uctx updatePlasmidContext,
) IOE.IOEither[error, *stock.PlasmidUpdate] {
	return func() E.Either[error, *stock.PlasmidUpdate] {
		if err := uctx.request.Validate(); err != nil {
			return E.Left[*stock.PlasmidUpdate](
				fmt.Errorf("invalid request parameters: %w", err),
			)
		}
		return E.Right[error](uctx.request)
	}
}

// updatePlasmidInRepository updates plasmid in repository
func updatePlasmidInRepository(
	uctx withValidatedUpdate,
) IOE.IOEither[error, *model.StockDoc] {
	return F.Pipe1(
		uctx.repo.EditPlasmid(uctx.validatedRequest),
		IOE.MapLeft[*model.StockDoc](
			fperrors.OnError("failed to update plasmid in repository"),
		),
	)
}

// retrieveFullPlasmidDoc retrieves the full plasmid document after update
func retrieveFullPlasmidDoc(
	uctx withUpdatedPlasmidDoc,
) IOE.IOEither[error, *model.StockDoc] {
	return func() E.Either[error, *model.StockDoc] {
		// Check if plasmid was found during update
		if uctx.stockDoc.NotFound {
			return E.Left[*model.StockDoc](
				fmt.Errorf("could not find plasmid with ID %s", uctx.stockDoc.ID),
			)
		}

		// Retrieve full plasmid to get all fields including ontology
		return uctx.repo.GetPlasmid(uctx.validatedRequest.Data.Id)()
	}
}

// transformToUpdatedPlasmidData transforms full stock document to plasmid data
var transformToUpdatedPlasmidData = F.Flow2(
	func(uctx withFullPlasmidDoc) *model.StockDoc { return uctx.fullStockDoc },
	makePlasmidData,
)

// publishUpdatedPlasmid publishes the updated plasmid event
func publishUpdatedPlasmid(
	uctx withUpdatedPlasmidData,
) IOE.IOEither[error, withUpdatedPlasmidData] {
	return IOE.TryCatchError(
		func() (withUpdatedPlasmidData, error) {
			plasmid := &stock.Plasmid{Data: uctx.plasmidData}
			err := uctx.publisher.PublishPlasmid(
				uctx.topics["stockUpdate"],
				plasmid,
			)
			if err != nil {
				return uctx, fmt.Errorf("failed to publish plasmid update event: %w", err)
			}
			return uctx, nil
		},
	)
}

// toUpdatePlasmidResult converts IOEither result to service response tuple
func toUpdatePlasmidResult(ctx context.Context) PlasmidConverter {
	return func(ioe PlasmidIO) PlasmidResult {
		return F.Pipe1(
			ioe(),
			E.Fold(
				func(e error) PlasmidResult {
					// Check if it's a not found error
					if isNotFoundError(e) {
						return T.MakeTuple2(
							&stock.Plasmid{},
							aphgrpc.HandleNotFoundError(ctx, e),
						)
					}
					return T.MakeTuple2(
						&stock.Plasmid{},
						aphgrpc.HandleUpdateError(ctx, e),
					)
				},
				func(plasmid *stock.Plasmid) PlasmidResult {
					return T.MakeTuple2[*stock.Plasmid, error](
						plasmid,
						nil,
					)
				},
			),
		)
	}
}

// LoadPlasmid workflow functions

// Curried setters for LoadPlasmid context building
var (
	// setValidatedExistingPlasmid sets validated existing plasmid request in context
	setValidatedExistingPlasmid = F.Curry2(
		func(params T.Tuple2[*stock.ExistingPlasmid, string], lctx loadPlasmidContext) withValidatedExistingPlasmid {
			return withValidatedExistingPlasmid{
				loadPlasmidContext: lctx,
				validatedRequest:   params.F1,
				plasmidID:          params.F2,
			}
		},
	)

	// setLoadedPlasmidDoc sets loaded stock document in context
	setLoadedPlasmidDoc = F.Curry2(
		func(doc *model.StockDoc, lctx withValidatedExistingPlasmid) withLoadedPlasmidDoc {
			return withLoadedPlasmidDoc{
				withValidatedExistingPlasmid: lctx,
				stockDoc:                     doc,
			}
		},
	)

	// setLoadedPlasmidData sets loaded plasmid data in context
	setLoadedPlasmidData = F.Curry2(
		func(data *stock.Plasmid_Data, lctx withLoadedPlasmidDoc) withLoadedPlasmidData {
			return withLoadedPlasmidData{
				withLoadedPlasmidDoc: lctx,
				plasmidData:          data,
			}
		},
	)

	// extractLoadPlasmidResponse extracts plasmid response from enriched context
	extractLoadPlasmidResponse = func(lctx withLoadedPlasmidData) *stock.Plasmid {
		return &stock.Plasmid{Data: lctx.plasmidData}
	}
)

// validateExistingPlasmidRequest validates the existing plasmid request
func validateExistingPlasmidRequest(
	lctx loadPlasmidContext,
) IOE.IOEither[error, T.Tuple2[*stock.ExistingPlasmid, string]] {
	return func() E.Either[error, T.Tuple2[*stock.ExistingPlasmid, string]] {
		if err := lctx.request.Validate(); err != nil {
			return E.Left[T.Tuple2[*stock.ExistingPlasmid, string]](
				fmt.Errorf("invalid request parameters: %w", err),
			)
		}

		// Apply default plasmid term if not provided
		if len(lctx.request.Data.Attributes.DictyPlasmidProperty) == 0 {
			lctx.request.Data.Attributes.DictyPlasmidProperty = lctx.params["plasmid_term"]
		}

		plasmidID := lctx.request.Data.Id

		return E.Right[error](T.MakeTuple2(lctx.request, plasmidID))
	}
}

// loadPlasmidInRepository loads plasmid in repository
func loadPlasmidInRepository(
	lctx withValidatedExistingPlasmid,
) IOE.IOEither[error, *model.StockDoc] {
	return F.Pipe1(
		lctx.repo.LoadPlasmid(lctx.plasmidID, lctx.validatedRequest),
		IOE.MapLeft[*model.StockDoc](
			fperrors.OnError("failed to load plasmid in repository"),
		),
	)
}

// transformToLoadedPlasmidData transforms stock document to plasmid data with ontology
var transformToLoadedPlasmidData = func(lctx withLoadedPlasmidDoc) *stock.Plasmid_Data {
	data := makePlasmidData(lctx.stockDoc)

	// Include ontology property if available
	if lctx.stockDoc.PlasmidProperties != nil {
		data.Attributes.DictyPlasmidProperty = lctx.stockDoc.PlasmidProperties.DictyPlasmidProperty
	}

	return data
}

// publishLoadedPlasmid publishes the loaded plasmid event
func publishLoadedPlasmid(
	lctx withLoadedPlasmidData,
) IOE.IOEither[error, withLoadedPlasmidData] {
	return IOE.TryCatchError(
		func() (withLoadedPlasmidData, error) {
			plasmid := &stock.Plasmid{Data: lctx.plasmidData}
			err := lctx.publisher.PublishPlasmid(
				lctx.topics["stockCreate"],
				plasmid,
			)
			if err != nil {
				return lctx, fmt.Errorf("failed to publish plasmid load event: %w", err)
			}
			return lctx, nil
		},
	)
}

// toLoadPlasmidResult converts IOEither result to service response tuple
func toLoadPlasmidResult(ctx context.Context) PlasmidConverter {
	return func(ioe PlasmidIO) PlasmidResult {
		return F.Pipe1(
			ioe(),
			E.Fold(
				func(e error) PlasmidResult {
					return T.MakeTuple2(
						&stock.Plasmid{},
						aphgrpc.HandleInsertError(ctx, e),
					)
				},
				func(plasmid *stock.Plasmid) PlasmidResult {
					return T.MakeTuple2[*stock.Plasmid, error](
						plasmid,
						nil,
					)
				},
			),
		)
	}
}

// isNotFoundError checks if an error is a "not found" error
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return len(errMsg) >= 23 && errMsg[:23] == "could not find plasmid"
}
