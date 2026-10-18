package arangodb

import (
	"context"

	fperrors "github.com/IBM/fp-go/errors"
	F "github.com/IBM/fp-go/function"
	IOE "github.com/IBM/fp-go/ioeither"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/model"
)

// ListPlasmids provides a list of all plasmids returning IOEither monad
func (ar *arangorepository) ListPlasmids(
	ctx context.Context,
	params *stock.StockParameters,
) IOE.IOEither[error, []*model.StockDoc] {
	baseCtx := normalizeQueryContext(ctx)
	withCtx := withPlasmidListQueryCtx(baseCtx)
	return F.Pipe5(
		IOE.Right[error](params),
		IOE.Map[error](ar.buildPlasmidListQueryParams),
		IOE.Map[error](withCtx),
		IOE.Chain(ar.executePlasmidListQuery),
		IOE.Chain(ar.scanPlasmidRows),
		IOE.MapLeft[[]*model.StockDoc](
			fperrors.OnError("failed to list plasmids"),
		),
	)
}

// GetPlasmid retrieves a plasmid from the database using IOEither monad
func (ar *arangorepository) GetPlasmid(
	id string,
) IOE.IOEither[error, *model.StockDoc] {
	return F.Pipe4(
		IOE.Of[error](id),
		IOE.Map[error](ar.buildPlasmidQueryParams),
		IOE.Chain(ar.executePlasmidQuery),
		IOE.Chain(ar.validatePlasmidQueryResult),
		IOE.MapLeft[*model.StockDoc](
			fperrors.OnError("failed to get plasmid"),
		),
	)
}
