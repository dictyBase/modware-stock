package statement

// StockCountsQ computes the total and subtype counts for strains and
// plasmids in one AQL statement. Each stock resolves its ontology term
// with the same FIRST(...) traversal over @stock_cvterm_graph used by
// FullSearchMerge, then maps that label to its subtype group:
//   - strain: "REMI-seq" -> "gwdi", "bacterial strain" -> "bacterial",
//     everything else -> "regular"
//   - plasmid: "GB vector" -> "goldenbraid", everything else -> "regular"
//
// Required bind variables:
//   - @@stock_collection
//   - @stock_prop_graph
//   - @stock_cvterm_graph
//   - @@cv_collection
//   - @strain_ontology
//   - @plasmid_ontology
//   - @gwdi_strain_tag
//   - @bacterial_strain_tag
//   - @goldenbraid_plasmid_tag
const StockCountsQ = `
LET typed_stocks = (
  FOR s IN @@stock_collection
    LET kind = FIRST(
      FOR prop, e IN 1..1 OUTBOUND s GRAPH @stock_prop_graph
        FILTER e.type IN ["strain", "plasmid"]
        RETURN e.type
    )
    FILTER kind != null
    LET term = FIRST(
      FOR cg IN 1..1 OUTBOUND s GRAPH @stock_cvterm_graph
        FOR cv IN @@cv_collection
          FILTER cg.deprecated == false
          FILTER cg.graph_id == cv._id
          FILTER cv.metadata.namespace == (kind == "strain" ? @strain_ontology : @plasmid_ontology)
          RETURN cg.label
    )
    LET subtype = kind == "strain"
      ? (term == @gwdi_strain_tag
          ? "gwdi"
          : (term == @bacterial_strain_tag ? "bacterial" : "regular"))
      : (term == @goldenbraid_plasmid_tag ? "goldenbraid" : "regular")
    RETURN {
      kind: kind,
      subtype: subtype
    }
)
LET strain_counts = (
  FOR x IN typed_stocks
    FILTER x.kind == "strain"
    COLLECT name = x.subtype WITH COUNT INTO count
    RETURN { name: name, count: count }
)
LET plasmid_counts = (
  FOR x IN typed_stocks
    FILTER x.kind == "plasmid"
    COLLECT name = x.subtype WITH COUNT INTO count
    RETURN { name: name, count: count }
)
LET strain_subtypes = (
  FOR name IN ["regular", "gwdi", "bacterial"]
    LET matched = FIRST(
      FOR c IN strain_counts
        FILTER c.name == name
        RETURN c.count
    )
    RETURN { name: name, count: NOT_NULL(matched, 0) }
)
LET plasmid_subtypes = (
  FOR name IN ["regular", "goldenbraid"]
    LET matched = FIRST(
      FOR c IN plasmid_counts
        FILTER c.name == name
        RETURN c.count
    )
    RETURN { name: name, count: NOT_NULL(matched, 0) }
)
RETURN {
  strains: {
    total: SUM(strain_subtypes[*].count),
    subtypes: strain_subtypes
  },
  plasmids: {
    total: SUM(plasmid_subtypes[*].count),
    subtypes: plasmid_subtypes
  }
}
`
