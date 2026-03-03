package service

import (
	"context"
	"fmt"
	"strings"

	E "github.com/IBM/fp-go/either"
	fperrors "github.com/IBM/fp-go/errors"
	F "github.com/IBM/fp-go/function"
	IOE "github.com/IBM/fp-go/ioeither"
	O "github.com/IBM/fp-go/option"
	P "github.com/IBM/fp-go/predicate"
	S "github.com/IBM/fp-go/string"
	T "github.com/IBM/fp-go/tuple"
	"github.com/dictyBase/aphgrpc"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/model"
)

var (
	// -- Shared Predicates & Helpers --

	// Use fp-go string API directly
	isNonEmptyString = S.IsNonEmpty

	isNotFoundError = F.Pipe1(isNotNilError, P.And(hasNotFoundPrefix))

	hasEnoughResults = F.Pipe1(
		hasMinimumResults,
		P.And(hasAnyCollectionResults),
	)

	shouldTrimLastItem = F.Pipe1(
		hasPositiveNextCursor,
		P.And(hasCollectionItems),
	)

	selectCollectionData = F.Ternary(
		shouldTrimLastItem,
		F.Flow2(extractCollectionData, trimLastCollectionItem),
		extractCollectionData,
	)

	// -- GetPlasmid Workflow --

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

	// -- ListPlasmids Workflow --

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

	// -- CreatePlasmid Workflow --

	// setValidatedNewPlasmid sets validated new plasmid request in context
	setValidatedNewPlasmid = F.Curry2(
		func(
			req *stock.NewPlasmid,
			cctx createPlasmidContext,
		) withValidatedNewPlasmid {
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

	applyDefaultPlasmidProperty = F.Curry2(
		func(defaultTerm string, req *stock.NewPlasmid) *stock.NewPlasmid {
			prop := F.Pipe2(
				req.Data.Attributes.DictyPlasmidProperty,
				O.FromPredicate(isNonEmptyString),
				O.GetOrElse(F.Constant(defaultTerm)),
			)
			req.Data.Attributes.DictyPlasmidProperty = prop
			return req
		},
	)

	// -- UpdatePlasmid Workflow --

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

	// -- LoadPlasmid Workflow --

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

	applyDefaultExistingPlasmidProperty = F.Curry2(
		func(defaultTerm string, req *stock.ExistingPlasmid) *stock.ExistingPlasmid {
			prop := F.Pipe2(
				req.Data.Attributes.DictyPlasmidProperty,
				O.FromPredicate(isNonEmptyString),
				O.GetOrElse(F.Constant(defaultTerm)),
			)
			req.Data.Attributes.DictyPlasmidProperty = prop
			return req
		},
	)
)

// validatePlasmidRequest validates the stock ID request
func validatePlasmidRequest(
	pctx getPlasmidContext,
) IOE.IOEither[error, string] {
	return F.Pipe2(
		IOE.TryCatchError(func() (*stock.StockId, error) {
			return pctx.request, pctx.request.Validate()
		}),
		IOE.MapLeft[*stock.StockId](
			fperrors.OnError("invalid request parameters"),
		),
		IOE.Map[error](func(req *stock.StockId) string { return req.Id }),
	)
}

// retrievePlasmidFromRepository retrieves plasmid from repository
func retrievePlasmidFromRepository(
	pctx withValidatedRequest,
) IOE.IOEither[error, *model.StockDoc] {
	return F.Pipe1(
		pctx.repo.GetPlasmid(pctx.validatedID),
		IOE.MapLeft[*model.StockDoc](
			fperrors.OnError(
				fmt.Sprintf(
					"failed to retrieve plasmid %s",
					pctx.validatedID,
				),
			),
		),
	)
}

func getStockDoc(pctx withStockDocument) *model.StockDoc {
	return pctx.stockDoc
}

func extractPlasmidResponse(data *stock.Plasmid_Data) *stock.Plasmid {
	return &stock.Plasmid{Data: data}
}

// toServiceResult converts IOEither result to service response tuple with error handling
func toServiceResult(ctx context.Context) PlasmidConverter {
	return F.Flow2(
		toEither[error, *stock.Plasmid],
		E.Fold(
			func(err error) PlasmidResult {
				return T.MakeTuple2(
					&stock.Plasmid{},
					aphgrpc.HandleGetError(ctx, err),
				)
			},
			func(plasmid *stock.Plasmid) PlasmidResult {
				return T.MakeTuple2[*stock.Plasmid, error](plasmid, nil)
			},
		),
	)
}

// ListPlasmids workflow functions

func hasPositiveNextCursor(
	ctx withNextCursor,
) bool {
	return ctx.nextCursor > 0
}

func hasCollectionItems(
	ctx withNextCursor,
) bool {
	return len(ctx.collectionData) > 0
}

func extractCollectionData(ctx withNextCursor) []*stock.PlasmidCollection_Data {
	return ctx.collectionData
}

func trimLastCollectionItem(
	data []*stock.PlasmidCollection_Data,
) []*stock.PlasmidCollection_Data {
	return data[:len(data)-1]
}

func extractPlasmidCollectionResponse(
	lctx withNextCursor,
) *stock.PlasmidCollection {
	pdata := F.Pipe1(lctx, selectCollectionData)
	return &stock.PlasmidCollection{
		Data: pdata,
		Meta: &stock.Meta{
			Limit:      lctx.limit,
			Total:      int64(len(pdata)),
			NextCursor: lctx.nextCursor,
		},
	}
}

// validatePlasmidFilter validates and processes the filter parameter
func validatePlasmidFilter(
	lctx listPlasmidsContext,
) IOE.IOEither[error, string] {
	return F.Pipe3(
		lctx.param.Filter,
		stockAQLStatementEither,
		E.MapLeft[string](fperrors.OnError("invalid filter parameter")),
		IOE.FromEither[error, string],
	)
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
func transformToPlasmidCollection(lctx withStockDocList) []*stock.PlasmidCollection_Data {
	return F.Pipe1(lctx.stockDocs, plasmidModelToCollectionSlice)
}

// computeNextCursor computes the next cursor value based on results
func hasMinimumResults(lctx withPlasmidCollectionData) bool {
	return len(lctx.collectionData) > int(lctx.limit)
}

func hasAnyCollectionResults(lctx withPlasmidCollectionData) bool {
	return len(lctx.collectionData) > 0
}

func lastItemCursorVal(lctx withPlasmidCollectionData) int64 {
	pdata := lctx.collectionData
	return genNextCursorVal(pdata[len(pdata)-1].Attributes.CreatedAt)
}

func computeNextCursor(lctx withPlasmidCollectionData) int64 {
	return F.Pipe1(
		lctx,
		F.Ternary(
			hasEnoughResults,
			lastItemCursorVal,
			F.Constant1[withPlasmidCollectionData](int64(0)),
		),
	)
}

// toPlasmidCollectionResult converts IOEither result to collection response tuple
func toPlasmidCollectionResult(
	ctx context.Context,
	limit int64,
) PlasmidCollectionConverter {
	return F.Flow2(
		toEither[error, *stock.PlasmidCollection],
		E.Fold(
			func(err error) PlasmidCollectionResult {
				return T.MakeTuple2(
					&stock.PlasmidCollection{Meta: &stock.Meta{Limit: limit}},
					aphgrpc.HandleGetError(ctx, err),
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

// CreatePlasmid workflow functions

// validateNewPlasmidRequest validates the new plasmid request
func validateNewPlasmidRequest(
	cctx createPlasmidContext,
) IOE.IOEither[error, *stock.NewPlasmid] {
	return F.Pipe2(
		IOE.TryCatchError(func() (*stock.NewPlasmid, error) {
			return cctx.request, cctx.request.Validate()
		}),
		IOE.MapLeft[*stock.NewPlasmid](
			fperrors.OnError("invalid request parameters"),
		),
		IOE.Map[error](
			applyDefaultPlasmidProperty(cctx.params["plasmid_term"]),
		),
	)
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
func transformToCreatedPlasmidData(cctx withCreatedPlasmidDoc) *stock.Plasmid_Data {
	return F.Pipe1(cctx.stockDoc, makePlasmidData)
}

// publishCreatedPlasmid publishes the created plasmid event
func publishCreatedPlasmid(
	cctx withCreatedPlasmidData,
) IOE.IOEither[error, withCreatedPlasmidData] {
	return F.Pipe1(
		IOE.TryCatchError(func() (withCreatedPlasmidData, error) {
			return cctx, cctx.publisher.PublishPlasmid(
				cctx.topics["stockCreate"],
				&stock.Plasmid{Data: cctx.plasmidData},
			)
		}),
		IOE.MapLeft[withCreatedPlasmidData](
			fperrors.OnError("failed to publish plasmid creation event"),
		),
	)
}

// toCreatePlasmidResult converts IOEither result to service response tuple
func toCreatePlasmidResult(ctx context.Context) PlasmidConverter {
	return F.Flow2(
		toEither[error, *stock.Plasmid],
		E.Fold(
			func(err error) PlasmidResult {
				return T.MakeTuple2(
					&stock.Plasmid{},
					aphgrpc.HandleInsertError(ctx, err),
				)
			},
			func(plasmid *stock.Plasmid) PlasmidResult {
				return T.MakeTuple2[*stock.Plasmid, error](plasmid, nil)
			},
		),
	)
}

// UpdatePlasmid workflow functions

// validateUpdatePlasmidRequest validates the update plasmid request
func validateUpdatePlasmidRequest(
	uctx updatePlasmidContext,
) IOE.IOEither[error, *stock.PlasmidUpdate] {
	return F.Pipe1(
		IOE.TryCatchError(func() (*stock.PlasmidUpdate, error) {
			return uctx.request, uctx.request.Validate()
		}),
		IOE.MapLeft[*stock.PlasmidUpdate](
			fperrors.OnError("invalid request parameters"),
		),
	)
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

func isStockDocNotFound(doc *model.StockDoc) bool { return doc.NotFound }

func stockDocNotFoundError(doc *model.StockDoc) error {
	return fmt.Errorf("could not find plasmid with ID %s", doc.ID)
}

// retrieveFullPlasmidDoc retrieves the full plasmid document after update
func retrieveFullPlasmidDoc(
	uctx withUpdatedPlasmidDoc,
) IOE.IOEither[error, *model.StockDoc] {
	return F.Pipe3(
		uctx.stockDoc,
		E.FromPredicate(P.Not(isStockDocNotFound), stockDocNotFoundError),
		IOE.FromEither[error, *model.StockDoc],
		IOE.Chain(func(_ *model.StockDoc) IOE.IOEither[error, *model.StockDoc] {
			return uctx.repo.GetPlasmid(uctx.validatedRequest.Data.Id)
		}),
	)
}

// transformToUpdatedPlasmidData transforms full stock document to plasmid data
func transformToUpdatedPlasmidData(uctx withFullPlasmidDoc) *stock.Plasmid_Data {
	return F.Pipe1(uctx.fullStockDoc, makePlasmidData)
}

// publishUpdatedPlasmid publishes the updated plasmid event
func publishUpdatedPlasmid(
	uctx withUpdatedPlasmidData,
) IOE.IOEither[error, withUpdatedPlasmidData] {
	return F.Pipe1(
		IOE.TryCatchError(func() (withUpdatedPlasmidData, error) {
			return uctx, uctx.publisher.PublishPlasmid(
				uctx.topics["stockUpdate"],
				&stock.Plasmid{Data: uctx.plasmidData},
			)
		}),
		IOE.MapLeft[withUpdatedPlasmidData](
			fperrors.OnError("failed to publish plasmid update event"),
		),
	)
}

func isNotNilError(err error) bool { return err != nil }

func hasNotFoundPrefix(err error) bool {
	return strings.HasPrefix(err.Error(), "could not find plasmid")
}

// toUpdatePlasmidResult converts IOEither result to service response tuple
func toUpdatePlasmidResult(ctx context.Context) PlasmidConverter {
	return F.Flow2(
		toEither[error, *stock.Plasmid],
		E.Fold(
			F.Ternary(
				isNotFoundError,
				func(err error) PlasmidResult {
					return T.MakeTuple2(
						&stock.Plasmid{},
						aphgrpc.HandleNotFoundError(ctx, err),
					)
				},
				func(err error) PlasmidResult {
					return T.MakeTuple2(
						&stock.Plasmid{},
						aphgrpc.HandleUpdateError(ctx, err),
					)
				},
			),
			func(plasmid *stock.Plasmid) PlasmidResult {
				return T.MakeTuple2[*stock.Plasmid, error](plasmid, nil)
			},
		),
	)
}

// LoadPlasmid workflow functions

// validateExistingPlasmidRequest validates the existing plasmid request
func validateExistingPlasmidRequest(
	lctx loadPlasmidContext,
) IOE.IOEither[error, T.Tuple2[*stock.ExistingPlasmid, string]] {
	return F.Pipe2(
		IOE.TryCatchError(func() (*stock.ExistingPlasmid, error) {
			return lctx.request, lctx.request.Validate()
		}),
		IOE.MapLeft[*stock.ExistingPlasmid](
			fperrors.OnError("invalid request parameters"),
		),
		IOE.Map[error](F.Flow2(
			applyDefaultExistingPlasmidProperty(lctx.params["plasmid_term"]),
			func(req *stock.ExistingPlasmid) T.Tuple2[*stock.ExistingPlasmid, string] {
				return T.MakeTuple2(req, req.Data.Id)
			},
		)),
	)
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
func transformToLoadedPlasmidData(
	lctx withLoadedPlasmidDoc,
) *stock.Plasmid_Data {
	data := makePlasmidData(lctx.stockDoc)
	prop := F.Pipe1(
		O.FromNillable(lctx.stockDoc.PlasmidProperties),
		O.Fold(
			F.Constant(data.Attributes.DictyPlasmidProperty),
			func(props *model.PlasmidProperties) string { return props.DictyPlasmidProperty },
		),
	)
	data.Attributes.DictyPlasmidProperty = prop
	return data
}

// publishLoadedPlasmid publishes the loaded plasmid event
func publishLoadedPlasmid(
	lctx withLoadedPlasmidData,
) IOE.IOEither[error, withLoadedPlasmidData] {
	return F.Pipe1(
		IOE.TryCatchError(func() (withLoadedPlasmidData, error) {
			return lctx, lctx.publisher.PublishPlasmid(
				lctx.topics["stockCreate"],
				&stock.Plasmid{Data: lctx.plasmidData},
			)
		}),
		IOE.MapLeft[withLoadedPlasmidData](
			fperrors.OnError("failed to publish plasmid load event"),
		),
	)
}

// toLoadPlasmidResult converts IOEither result to service response tuple
func toLoadPlasmidResult(ctx context.Context) PlasmidConverter {
	return F.Flow2(
		toEither[error, *stock.Plasmid],
		E.Fold(
			func(err error) PlasmidResult {
				return T.MakeTuple2(
					&stock.Plasmid{},
					aphgrpc.HandleInsertError(ctx, err),
				)
			},
			func(plasmid *stock.Plasmid) PlasmidResult {
				return T.MakeTuple2[*stock.Plasmid, error](plasmid, nil)
			},
		),
	)
}
