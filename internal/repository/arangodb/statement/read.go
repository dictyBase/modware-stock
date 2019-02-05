package statement

const (
	StockFindIdQ = `
		FOR v IN 1..1 OUTBOUND
			CONCAT(@@stock_collection,"/",@stock_id) GRAPH @graph
			RETURN { propkey: v._key }
	`
	StockFindQ = `
		FOR s IN @@stock_collection
			FILTER s.stock_id == @id
			LIMIT 1
			RETURN s._id
	`
	StockGetStrain = `
		FOR s IN @@stock_collection
			FOR v, e IN 1..1 OUTBOUND s GRAPH @graph
				FILTER e.type == 'strain'
				FILTER s.stock_id == @id
				RETURN MERGE(
					s,
					{
						strain_properties: {
							systematic_name: v.systematic_name,
							label: v.label,
							species: v.species,
							plasmid: v.plasmid,
							parent: v.parent,
							names: v.names
						}
					}
				)
	`
	StockGetPlasmid = `
		FOR s IN @@stock_collection
			FOR v, e IN 1..1 OUTBOUND s GRAPH @graph
				FILTER e.type == 'plasmid'
				FILTER s.stock_id == @id
				RETURN MERGE(
					s,
					{
						plasmid_properties: {
							image_map: v.image_map,
							sequence: v.sequence
						}
					}
				)
	`
	StockList = `
		FOR stock IN @@stock_collection
			SORT stock.created_at DESC
			LIMIT @limit
			RETURN stock
	`
	StockListWithCursor = `
		FOR stock in @@stock_collection
			FILTER stock.created_at <= DATE_ISO8601(@next_cursor)
			SORT stock.created_at DESC
			LIMIT @limit
			RETURN stock
	`
	StrainGetParentRel = `
		FOR v,e IN 1..1 INBOUND @strain_key GRAPH @parent_graph
			RETURN e._key
	`
	StrainList = `
		FOR s IN @@stock_collection
			FOR v IN 1..1 OUTBOUND s GRAPH @graph
				SORT s.created_at DESC
				LIMIT @limit
				RETURN MERGE(
					s,
					{
						strain_properties: {
							systematic_name: v.systematic_name,
							label: v.label,
							species: v.species,
							plasmid: v.plasmid,
							parents: v.parents,
							names: v.names
						}
					}
				)
	`
	StrainListWithCursor = `
		FOR s IN @@stock_collection
			FOR v IN 1..1 OUTBOUND s GRAPH @graph
				FILTER s.created_at <= DATE_ISO8601(@next_cursor)
				SORT s.created_at DESC
				LIMIT @limit
				RETURN MERGE(
					s,
					{
						strain_properties: {
							systematic_name: v.systematic_name,
							label: v.label,
							species: v.species,
							plasmid: v.plasmid,
							parents: v.parents,
							names: v.names
						}
					}
				)
	`
	PlasmidList = `
		FOR s IN @@stock_collection
			FOR v IN 1..1 OUTBOUND s GRAPH @graph
				SORT s.created_at DESC
				LIMIT @limit
				RETURN MERGE(
					s,
					{
						plasmid_properties: {
							image_map: v.image_map,
							sequence: v.sequence
						}
					}
				)
	`
	PlasmidListWithCursor = `
		FOR s IN @@stock_collection
			FOR v IN 1..1 OUTBOUND s GRAPH @graph
				FILTER s.created_at <= DATE_ISO8601(@next_cursor)
				SORT s.created_at DESC
				LIMIT @limit
				RETURN MERGE(
					s,
					{
						plasmid_properties: {
							image_map: v.image_map,
							sequence: v.sequence
						}
					}
				)
	`
)
