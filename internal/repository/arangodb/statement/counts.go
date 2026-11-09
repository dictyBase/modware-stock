package statement

// StockCountsQ computes the total and subtype counts for strains and
// plasmids in one AQL statement by aggregating the edge and ontology
// collections directly instead of running per-stock GRAPH traversals:
//   - strain: "REMI-seq" -> "gwdi", "bacterial strain" -> "bacterial",
//     everything else -> "regular"
//   - plasmid: "GB vector" -> "goldenbraid", everything else -> "regular"
//
// Required bind variables:
//   - @@stock_type_collection
//   - @@stock_term_collection
//   - @@cvterm_collection
//   - @@cv_collection
//   - @strain_ontology
//   - @plasmid_ontology
//   - @gwdi_strain_tag
//   - @bacterial_strain_tag
//   - @goldenbraid_plasmid_tag
const StockCountsQ = `
LET type_totals = (
  FOR e IN @@stock_type_collection
    FILTER e.type IN ["strain", "plasmid"]
    COLLECT kind = e.type WITH COUNT INTO total
    RETURN { kind: kind, total: total }
)
LET strain_total = NOT_NULL(
  FIRST(FOR t IN type_totals FILTER t.kind == "strain" RETURN t.total),
  0
)
LET plasmid_total = NOT_NULL(
  FIRST(FOR t IN type_totals FILTER t.kind == "plasmid" RETURN t.total),
  0
)
LET special_terms = (
  FOR cv IN @@cv_collection
    FILTER cv.metadata.namespace IN [@strain_ontology, @plasmid_ontology]
    FOR cg IN @@cvterm_collection
      FILTER cg.graph_id == cv._id AND cg.deprecated == false
      FILTER (cv.metadata.namespace == @strain_ontology AND cg.label IN [@gwdi_strain_tag, @bacterial_strain_tag])
          OR (cv.metadata.namespace == @plasmid_ontology AND cg.label == @goldenbraid_plasmid_tag)
      FOR st IN @@stock_term_collection
        FILTER st._to == cg._id
        FOR e IN @@stock_type_collection
          FILTER e._from == st._from
          FILTER (cv.metadata.namespace == @strain_ontology AND e.type == "strain")
              OR (cv.metadata.namespace == @plasmid_ontology AND e.type == "plasmid")
          COLLECT kind = e.type, label = cg.label WITH COUNT INTO count
          RETURN { kind: kind, label: label, count: count }
)
LET gwdi_count = NOT_NULL(
  FIRST(
    FOR x IN special_terms
      FILTER x.kind == "strain" AND x.label == @gwdi_strain_tag
      RETURN x.count
  ),
  0
)
LET bacterial_count = NOT_NULL(
  FIRST(
    FOR x IN special_terms
      FILTER x.kind == "strain" AND x.label == @bacterial_strain_tag
      RETURN x.count
  ),
  0
)
LET gb_count = NOT_NULL(
  FIRST(
    FOR x IN special_terms
      FILTER x.kind == "plasmid" AND x.label == @goldenbraid_plasmid_tag
      RETURN x.count
  ),
  0
)
RETURN {
  strains: {
    total: strain_total,
    subtypes: [
      { name: "regular", count: strain_total - gwdi_count - bacterial_count },
      { name: "gwdi", count: gwdi_count },
      { name: "bacterial", count: bacterial_count }
    ]
  },
  plasmids: {
    total: plasmid_total,
    subtypes: [
      { name: "regular", count: plasmid_total - gb_count },
      { name: "goldenbraid", count: gb_count }
    ]
  }
}
`
