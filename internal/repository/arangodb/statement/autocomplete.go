// Package statement holds the AQL statements of the arangodb
// repository.
package statement

// AutocompleteStockBranch is one branch of the autocomplete statement
// over the stock collection. The builder fills it with fmt.Sprintf in
// this verb order:
//
//  1. %s — the branch variable (p<n> for the prefix stage, n<n> for the
//     fuzzy stage)
//  2. %s — the SEARCH expression
//  3. %q — the field label that the branch reports
//  4. %s — the display expression
//  5. %s — the score expression
//
// It requires the bind parameters q, stock_prop_graph, entity, limit.
// The view name stock_autocomplete is written literally, because
// ArangoDB does not accept a bind parameter for a view name in SEARCH.
// The row key is the stock key, so the branch reads the entity from one
// OUTBOUND traversal over the stock property graph.
const AutocompleteStockBranch = `LET %s = (
  FOR d IN stock_autocomplete
    SEARCH %s
    LET ent = FIRST(
      FOR v, e IN 1..1 OUTBOUND d GRAPH @stock_prop_graph
        RETURN e.type
    )
    FILTER ent != null
    FILTER @entity == "" OR @entity == ent
    SORT BM25(d) DESC, d._key ASC
    LIMIT @limit
    RETURN {
      k: d._key,
      id: d.stock_id,
      entity: ent,
      f: %q,
      v: %s,
      s: %s
    }
)`

// AutocompletePropBranch is one branch of the autocomplete statement
// over the stock property collection. The builder fills it with
// fmt.Sprintf in the same verb order as AutocompleteStockBranch. It
// requires the same bind parameters. The property _key is
// auto-generated and carries no relation to the stock key, so the row
// resolves its owner with one INBOUND traversal.
const AutocompletePropBranch = `LET %s = (
  FOR d IN stock_autocomplete
    SEARCH %s
    LET own = FIRST(
      FOR v, e IN 1..1 INBOUND d GRAPH @stock_prop_graph
        RETURN { k: v._key, id: v.stock_id, entity: e.type }
    )
    FILTER own != null
    FILTER @entity == "" OR @entity == own.entity
    SORT BM25(d) DESC, own.k ASC
    LIMIT @limit
    RETURN {
      k: own.k,
      id: own.id,
      entity: own.entity,
      f: %q,
      v: %s,
      s: %s
    }
)`

// AutocompleteMerge merges the 16 branches of the autocomplete
// statement and returns the top rows. The builder fills it with one
// verb:
//
//  1. %s — the comma-separated list of the 16 branch variables
//
// It requires the bind parameter limit. The merge keeps one best row
// per stock key by score DESC and field label ASC, then sorts by score
// DESC and stock key ASC. The variable is named hits, because all is a
// reserved word in AQL; the collect variable is stkey.
const AutocompleteMerge = `LET hits = FLATTEN([%s])
LET best = (
  FOR x IN hits
    COLLECT stkey = x.k INTO grp = x
    LET top = FIRST(
      FOR m IN grp
        SORT m.s DESC, m.f ASC
        RETURN m
    )
    RETURN top
)
FOR x IN best
  SORT x.s DESC, x.k ASC
  LIMIT @limit
  RETURN {
    k: x.k,
    id: x.id,
    entity: x.entity,
    f: x.f,
    v: x.v,
    s: x.s
  }`
