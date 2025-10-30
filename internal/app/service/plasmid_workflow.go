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
