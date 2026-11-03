package arangodb

import (
	"context"

	A "github.com/IBM/fp-go/v2/array"
	fperrors "github.com/IBM/fp-go/v2/errors"
	F "github.com/IBM/fp-go/v2/function"
	IOE "github.com/IBM/fp-go/v2/ioeither"
	O "github.com/IBM/fp-go/v2/option"
	predarrays "github.com/dictyBase/fp-go-loom/predicate/array"
	"github.com/dictyBase/modware-stock/internal/repository"
	"github.com/dictyBase/modware-stock/internal/repository/arangodb/statement"
)

type (
	subtypeCountRow struct {
		Name  string `json:"name"`
		Count int64  `json:"count"`
	}

	entityCountsRow struct {
		Total    int64             `json:"total"`
		Subtypes []subtypeCountRow `json:"subtypes"`
	}

	stockCountsRow struct {
		Strains  entityCountsRow `json:"strains"`
		Plasmids entityCountsRow `json:"plasmids"`
	}

	stockCountsQueryParams struct {
		ctx        context.Context
		statement  string
		bindParams map[string]any
	}
)

const (
	paramGwdiStrainTag         = "gwdi_strain_tag"
	paramBacterialStrainTag    = "bacterial_strain_tag"
	paramGoldenBraidPlasmidTag = "goldenbraid_plasmid_tag"
	gwdiStrainTag              = "REMI-seq"
	bacterialStrainTag         = "bacterial strain"
	goldenBraidPlasmidTag      = "GB vector"

	// Subtype group names shared by the empty-counts fallback and the
	// counts tests. The AQL statement pins the same names.
	subtypeRegular     = "regular"
	subtypeGwdi        = "gwdi"
	subtypeBacterial   = "bacterial"
	subtypeGoldenBraid = "goldenbraid"
)

func (ar *arangorepository) buildStockCountsQueryParams(
	ctx context.Context,
) stockCountsQueryParams {
	return stockCountsQueryParams{
		ctx:       normalizeQueryContext(ctx),
		statement: statement.StockCountsQ,
		bindParams: map[string]any{
			bindStockCollection:        ar.stockc.stock.Name(),
			nameStockPropGraph:         ar.stockc.stockPropType.Name(),
			nameStockCvtermGraph:       ar.stockc.stockOnto.Name(),
			bindCVCollection:           ar.ontoc.Cv.Name(),
			paramStrainOntology:        ar.strainOnto,
			paramPlasmidOntology:       ar.plasmidOnto,
			paramGwdiStrainTag:         gwdiStrainTag,
			paramBacterialStrainTag:    bacterialStrainTag,
			paramGoldenBraidPlasmidTag: goldenBraidPlasmidTag,
		},
	}
}

func (ar *arangorepository) executeStockCountsQuery(
	params stockCountsQueryParams,
) IOE.IOEither[error, dbRows] {
	return F.Pipe1(
		IOE.TryCatchError(
			func() (dbRows, error) {
				return ar.searchRows(
					params.ctx,
					params.statement,
					params.bindParams,
				)
			},
		),
		IOE.MapLeft[dbRows](
			fperrors.OnError("error in running stock counts query"),
		),
	)
}

func mapSubtypeCountRow(row subtypeCountRow) repository.SubtypeCount {
	return repository.SubtypeCount{
		Name:  row.Name,
		Count: row.Count,
	}
}

func emptySubtypeCounts() []repository.SubtypeCount {
	return []repository.SubtypeCount{}
}

func mapEntityCountsRow(row entityCountsRow) repository.EntityCounts {
	return repository.EntityCounts{
		Total: row.Total,
		Subtypes: F.Pipe2(
			row.Subtypes,
			O.FromPredicate(predarrays.IsNonEmpty[subtypeCountRow]()),
			O.Fold(emptySubtypeCounts, A.Map(mapSubtypeCountRow)),
		),
	}
}

func emptyStockCounts() *repository.StockCounts {
	return &repository.StockCounts{
		Strains: repository.EntityCounts{
			Subtypes: []repository.SubtypeCount{
				{Name: subtypeRegular, Count: 0},
				{Name: subtypeGwdi, Count: 0},
				{Name: subtypeBacterial, Count: 0},
			},
		},
		Plasmids: repository.EntityCounts{
			Subtypes: []repository.SubtypeCount{
				{Name: subtypeRegular, Count: 0},
				{Name: subtypeGoldenBraid, Count: 0},
			},
		},
	}
}

func mapStockCountsRow(row stockCountsRow) *repository.StockCounts {
	return &repository.StockCounts{
		Strains:  mapEntityCountsRow(row.Strains),
		Plasmids: mapEntityCountsRow(row.Plasmids),
	}
}

// CountStocks returns the totals and subtype counts for strains and
// plasmids as a lazy IOEither pipeline.
func (ar *arangorepository) CountStocks(
	ctx context.Context,
) IOE.IOEither[error, *repository.StockCounts] {
	return F.Pipe5(
		IOE.Of[error](ctx),
		IOE.Map[error](ar.buildStockCountsQueryParams),
		IOE.Chain(ar.executeStockCountsQuery),
		IOE.Chain(
			scanSearchRows[stockCountsRow]("error in reading stock counts row"),
		),
		IOE.Map[error](A.Head[stockCountsRow]),
		IOE.Map[error](O.Fold(emptyStockCounts, mapStockCountsRow)),
	)
}
