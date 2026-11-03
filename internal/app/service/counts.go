package service

import (
	"context"

	A "github.com/IBM/fp-go/v2/array"
	E "github.com/IBM/fp-go/v2/either"
	F "github.com/IBM/fp-go/v2/function"
	IOE "github.com/IBM/fp-go/v2/ioeither"
	O "github.com/IBM/fp-go/v2/option"
	ioeutils "github.com/dictyBase/fp-go-loom/ioeitherutils"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/repository"
	empty "google.golang.org/protobuf/types/known/emptypb"
)

type (
	// countStocksInput is the boundary input of one stock count call:
	// the caller context and the repository it queries.
	countStocksInput struct {
		ctx  context.Context
		repo repository.StockRepository
	}

	// countStocksResult is the boundary record of one stock count call:
	// the proto response and the gRPC status error, exactly one of
	// which the handler returns.
	countStocksResult struct {
		counts *stock.StockCounts
		err    error
	}
)

func queryStockCounts(
	in countStocksInput,
) IOE.IOEither[error, *repository.StockCounts] {
	return F.Pipe1(
		in.repo.CountStocks(in.ctx),
		IOE.MapLeft[*repository.StockCounts](serverErrorStatus),
	)
}

func mapProtoSubtypeCount(
	sub repository.SubtypeCount,
) *stock.StockSubtypeCount {
	return &stock.StockSubtypeCount{
		Name:  sub.Name,
		Count: sub.Count,
	}
}

func mapProtoEntityCounts(
	ec repository.EntityCounts,
) *stock.StockEntityCounts {
	return &stock.StockEntityCounts{
		Total:    ec.Total,
		Subtypes: F.Pipe1(ec.Subtypes, A.Map(mapProtoSubtypeCount)),
	}
}

func emptyProtoStockCounts() *stock.StockCounts {
	return &stock.StockCounts{
		Strains:  &stock.StockEntityCounts{},
		Plasmids: &stock.StockEntityCounts{},
	}
}

func buildProtoStockCounts(
	counts repository.StockCounts,
) *stock.StockCounts {
	return &stock.StockCounts{
		Strains:  mapProtoEntityCounts(counts.Strains),
		Plasmids: mapProtoEntityCounts(counts.Plasmids),
	}
}

func failedCountStocksResult(err error) countStocksResult {
	return countStocksResult{err: err}
}

func successfulCountStocksResult(
	counts *stock.StockCounts,
) countStocksResult {
	return countStocksResult{counts: counts}
}

// CountStocks returns totals and subtype counts for all strains and
// plasmids.
func (s *StockService) CountStocks(
	ctx context.Context,
	_ *empty.Empty,
) (*stock.StockCounts, error) {
	result := F.Pipe5(
		IOE.Of[error](countStocksInput{ctx: ctx, repo: s.repo}),
		IOE.Chain(queryStockCounts),
		IOE.Map[error](O.FromNillable2[repository.StockCounts]),
		IOE.Map[error](O.Fold(emptyProtoStockCounts, buildProtoStockCounts)),
		ioeutils.ToEither[error, *stock.StockCounts],
		E.Fold(failedCountStocksResult, successfulCountStocksResult),
	)
	return result.counts, result.err
}
