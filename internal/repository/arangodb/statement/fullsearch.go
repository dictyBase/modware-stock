// Package statement holds the AQL statements of the arangodb
// repository.
package statement

const (
	// FullSearchStockBranch is one branch of the full search statement
	// over the stock collection. The builder fills it with fmt.Sprintf in
	// this verb order:
	//
	//  1. %s — the branch variable (p<n> for prefix, n<n> for fuzzy, t<n>
	//     for token, h<n> for phrase)
	//  2. %s — the SEARCH expression
	//  3. %q — the field label that the branch reports
	//  4. %s — the display expression
	//  5. %s — the score expression
	//
	// It requires the bind parameters q, stock_prop_graph, entity, limit.
	// The view name stock_full_search is written literally, because
	// ArangoDB does not accept a bind parameter for a view name in SEARCH.
	// The row key is the stock key, so one OUTBOUND traversal over the
	// stock property graph supplies the entity and the stock label.
	FullSearchStockBranch = `LET %s = (
  FOR d IN stock_full_search
    SEARCH %s
    LET meta = FIRST(
      FOR prop, edge IN 1..1 OUTBOUND d GRAPH @stock_prop_graph
        RETURN {
          entity: edge.type,
          stock_label: edge.type == "strain"
            ? NOT_NULL(prop.label, "")
            : NOT_NULL(prop.name, "")
        }
    )
    FILTER meta != null
    FILTER @entity == "" OR @entity == meta.entity
    SORT BM25(d) DESC, d._key ASC
    LIMIT @limit
    RETURN {
      k: d._key,
      id: d.stock_id,
      entity: meta.entity,
      sl: meta.stock_label,
      f: %q,
      v: %s,
      s: %s
    }
)`

	// FullSearchPropBranch is one branch of the full search statement over
	// the stock property collection. The builder fills it with fmt.Sprintf
	// in this verb order:
	//
	//  1. %s — the branch variable (p<n> for prefix, n<n> for fuzzy)
	//  2. %s — the SEARCH expression
	//  3. %q — the static entity type that owns the property field
	//     ("strain" or "plasmid")
	//  4. %q — the field label that the branch reports
	//  5. %s — the display expression
	//  6. %s — the score expression
	//
	// It requires the same bind parameters as FullSearchStockBranch. Each
	// property field belongs to one entity type, so the branch filters by
	// @entity, ranks, and caps the view rows before it resolves the owner
	// with one INBOUND traversal. A strain row carries d.label and a
	// plasmid row carries d.name as the stock label.
	FullSearchPropBranch = `LET %s = (
  FOR d IN stock_full_search
    SEARCH %s
    FILTER @entity == "" OR @entity == %q
    SORT BM25(d) DESC, d._key ASC
    LIMIT @limit
    LET own = FIRST(
      FOR v, e IN 1..1 INBOUND d GRAPH @stock_prop_graph
        RETURN {
          k: v._key,
          id: v.stock_id,
          entity: e.type,
          stock_label: e.type == "strain"
            ? NOT_NULL(d.label, "")
            : NOT_NULL(d.name, "")
        }
    )
    FILTER own != null
    FILTER @entity == "" OR @entity == own.entity
    RETURN {
      k: own.k,
      id: own.id,
      entity: own.entity,
      sl: own.stock_label,
      f: %q,
      v: %s,
      s: %s
    }
)`

	// FullSearchMerge merges the 19 branches of the full search statement
	// and returns the top rows with their complete stored summaries and
	// ontology term labels. The builder fills it with one verb:
	//
	//  1. %s — the comma-separated list of the 19 branch variables
	//
	// It requires the bind parameters limit, stock_collection,
	// stock_cvterm_graph, @cv_collection, strain_ontology, and
	// plasmid_ontology. The merge keeps one best row per stock key by
	// score DESC and field label ASC, then sorts by score DESC and stock
	// key ASC. The DOCUMENT lookup and ontology traversal sit after the
	// tail LIMIT, so they run at most 50 times per request. The variable
	// is named hits, because all is a reserved word in AQL; the collect
	// variable is stkey.
	FullSearchMerge = `LET hits = FLATTEN([%s])
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
  LET stk = DOCUMENT(@stock_collection, x.k)
  LET term = FIRST(
    FOR cg IN 1..1 OUTBOUND stk GRAPH @stock_cvterm_graph
      FOR cv IN @@cv_collection
        FILTER cg.deprecated == false
        FILTER cg.graph_id == cv._id
        FILTER cv.metadata.namespace == (x.entity == "strain" ? @strain_ontology : @plasmid_ontology)
        RETURN cg.label
  )
  RETURN {
    k: x.k,
    id: x.id,
    entity: x.entity,
    f: x.f,
    v: x.v,
    s: x.s,
    sm: NOT_NULL(stk.summary, ""),
    sl: x.sl,
    st: NOT_NULL(term, "")
  }`
)
