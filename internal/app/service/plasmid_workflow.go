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
