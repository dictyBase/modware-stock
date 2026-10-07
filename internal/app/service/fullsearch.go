package service

import (
	"context"
	"strings"

	A "github.com/IBM/fp-go/array"
	F "github.com/IBM/fp-go/function"
	"github.com/bufbuild/protovalidate-go"
	"github.com/cockroachdb/errors"
	"github.com/dictyBase/aphgrpc"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/repository"
)

// defaultFullSearchLimit is the effective limit for a request limit of
// 0. It mirrors the repository clamp.
const defaultFullSearchLimit = 50

// maxFullSearchLimit is the hard cap of the returned list length. The
// proto accepts a limit up to 100, and the handler clamps 51 to 100
// down to this value, because the spec asks for one list of 50 items.
const maxFullSearchLimit = 50

// SearchStock returns at most 50 ranked results for a search text of
// at least 2 characters. The order of operations is: the protovalidate
// validation, the trim guard, the effective limit, the entity mapping
// and the repository call. protovalidate rejects nil data and nil
// attributes at the boundary, because the proto marks both required,
// so no explicit nil guard is needed.
func (s *StockService) SearchStock(
	ctx context.Context,
	r *stock.StockSearchParameters,
) (*stock.StockSearchResultCollection, error) {
	// 1. The generated Validate method is a no-op for these messages,
	// and it also skips absent nested messages; protovalidate does not.
	if err := protovalidate.Validate(r); err != nil {
		return nil, aphgrpc.HandleInvalidParamError(ctx, err)
	}
	attrs := r.GetData().GetAttributes()
	// 2. The proto rule counts characters, so a whitespace-only query
	// reaches this point.
	query := strings.TrimSpace(attrs.GetQuery())
	if query == "" {
		return nil, aphgrpc.HandleInvalidParamError(
			ctx,
			errors.New("expect a non-empty search query"),
		)
	}
	// 3. A limit of 0 becomes 50, and a limit from 51 to 100 becomes 50.
	effLimit := attrs.GetLimit()
	if effLimit <= 0 {
		effLimit = defaultFullSearchLimit
	}
	if effLimit > maxFullSearchLimit {
		effLimit = maxFullSearchLimit
	}
	// 4. The entity mapping rejects a value outside the three defined
	// enum values, so a future proto change cannot open a silent hole.
	entity, err := autocompleteEntityFilter(attrs.GetEntity())
	if err != nil {
		return nil, aphgrpc.HandleInvalidParamError(ctx, err)
	}
	// 5. An empty result is not a not-found error.
	rows, err := s.repo.SearchStock(&repository.FullSearchQuery{
		Query:  query,
		Entity: entity,
		Limit:  int(effLimit),
	})
	if err != nil {
		return nil, aphgrpc.HandleGetError(ctx, err)
	}
	// 6. Every result carries the complete stored summary and the
	// strain label.
	return &stock.StockSearchResultCollection{
		Data: F.Pipe1(rows, A.Map(mapFullSearchResult)),
		Meta: &stock.Meta{
			NextCursor: 0,
			Limit:      effLimit,
			Total:      int64(len(rows)),
		},
	}, nil
}

// mapFullSearchResult converts one repository result into the proto
// message of the response.
func mapFullSearchResult(row *repository.FullSearchResult) *stock.StockSearchResult {
	return &stock.StockSearchResult{
		Id:          row.ID,
		Entity:      mapRepositoryEntity(row.Entity),
		Field:       mapRepositoryField(row.Field),
		DisplayText: row.DisplayText,
		Score:       row.Score,
		Summary:     row.Summary,
		StrainLabel: row.StrainLabel,
	}
}
