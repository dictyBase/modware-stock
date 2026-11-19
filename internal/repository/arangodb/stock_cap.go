package arangodb

import (
	F "github.com/IBM/fp-go/v2/function"
	O "github.com/IBM/fp-go/v2/option"
	S "github.com/IBM/fp-go/v2/string"
	"github.com/dictyBase/modware-stock/internal/repository"
)

// entityOverfetch multiplies the branch limit when an entity filter is
// set. The stock branches learn the entity of a row only from the
// traversal that follows the cap, so a filtered request reads more view
// rows to keep the filtered list near the requested length.
const entityOverfetch = 5

// entityFilterIsSet reports whether the request narrows the search to one
// entity.
func entityFilterIsSet(entity repository.StockEntityFilter) bool {
	return S.IsNonEmpty(string(entity))
}

// stockBranchCap is the number of view rows that a stock branch keeps
// before its owner traversal. It equals the limit without an entity
// filter, because then no row drops out after the traversal.
func stockBranchCap(limit int, entity repository.StockEntityFilter) int {
	return F.Pipe3(
		entity,
		O.FromPredicate(entityFilterIsSet),
		O.Map(F.Constant1[repository.StockEntityFilter](limit*entityOverfetch)),
		O.GetOrElse(F.Constant(limit)),
	)
}
