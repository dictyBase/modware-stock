package statement

const (
	StockFindIdQ = `
		FOR v IN 1..1 OUTBOUND
			CONCAT(@stock_collection,"/",@stock_id) GRAPH @graph
			RETURN v._key
	`
	StockFindQ = `
		FOR s IN @@stock_collection
			FILTER s.stock_id == @id
			LIMIT 1
			RETURN s._id
	`
	StockGetStrain = `
		FOR v, e IN 1..1 OUTBOUND CONCAT(@stock_collection,"/",@id) GRAPH @prop_graph
			LET parent = (
				FOR pg IN 1..1 INBOUND CONCAT(@stock_collection,"/",@id) GRAPH @parent_graph
					RETURN pg.stock_id
			)
			LET term = (
				FOR cg IN 1..1 OUTBOUND CONCAT(@stock_collection,"/",@id) GRAPH @stock_cvterm_graph
					FOR cv IN @@cv_collection
						FILTER cg.deprecated == false
						FILTER cg.graph_id == cv._id
						FILTER cv.metadata.namespace == @ontology
						RETURN cg.label
			)
			FILTER e.type == 'strain'
			FOR s IN @@stock_collection
				FILTER s.stock_id == @id
				LIMIT 1
				RETURN MERGE(s, {
					strain_properties: {
							label: v.label,
							species: v.species,
							plasmid: v.plasmid,
							names: v.names,
							dicty_strain_property: term[0],
							parent: parent[0]
			}})
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
							sequence: v.sequence,
							name: v.name
						}
					}
				)
	`
	StrainGetParentRel = `
		FOR v,e IN 1..1 INBOUND @strain_key GRAPH @parent_graph
			RETURN e._key
	`

	StrainListFromIds = `
		FOR id IN @ids
			FOR v, e IN 1..1 OUTBOUND CONCAT(@stock_collection,"/",id) GRAPH @prop_graph
				LET parent = (
							FOR p IN 1..1 INBOUND 
										CONCAT(@stock_collection,"/",id) 
										GRAPH @parent_graph
								RETURN p.stock_id
								
				)
				FILTER e.type == 'strain'
				FOR s IN @@stock_collection
					FILTER s.stock_id == id
					LIMIT @limit
					RETURN MERGE(s,{
							strain_properties: {
								parent: parent[0],
								label: v.label,
								species: v.species,
								plasmid: v.plasmid,
								names: v.names
						}
					})
	`

	StrainList = `
		FOR s IN %s
			FOR v, e IN 1..1 OUTBOUND s GRAPH '%s'
				FILTER e.type == 'strain'
				SORT s.created_at DESC
				LIMIT %d
				RETURN MERGE(
					s,
					{
						strain_properties: { 
							label: v.label, 
							species: v.species, 
							plasmid: v.plasmid, 
							names: v.names
						} 
					}
				)
	`
	StrainListFilter = `
		FOR s IN %s
			FOR v, e IN 1..1 OUTBOUND s GRAPH '%s'
				FILTER e.type == 'strain'
				%s
				SORT s.created_at DESC
				LIMIT %d
				RETURN MERGE(
					s,
					{
						strain_properties: { 
							label: v.label, 
							species: v.species, 
							plasmid: v.plasmid, 
							names: v.names
						} 
					}
				)
	`
	StrainListWithCursor = `
		FOR s IN %s
			FOR v, e IN 1..1 OUTBOUND s GRAPH '%s'
				FILTER e.type == 'strain'
				FILTER s.created_at <= DATE_ISO8601(%d)
				SORT s.created_at DESC
				LIMIT %d
				RETURN MERGE(
					s,
					{
						strain_properties: { 
							label: v.label, 
							species: v.species, 
							plasmid: v.plasmid, 
							names: v.names
						} 
					}
				)
	`
	StrainListFilterWithCursor = `
		FOR s IN %s
			FOR v, e IN 1..1 OUTBOUND s GRAPH '%s'
				FILTER e.type == 'strain'
				%s
				FILTER s.created_at <= DATE_ISO8601(%d)
				SORT s.created_at DESC
				LIMIT %d
				RETURN MERGE(
					s,
					{
						strain_properties: { 
							label: v.label, 
							species: v.species, 
							plasmid: v.plasmid, 
							names: v.names
						} 
					}
				)
	`
	PlasmidList = `
		FOR s IN %s
			FOR v, e IN 1..1 OUTBOUND s GRAPH '%s'
				FILTER e.type == 'plasmid'
				SORT s.created_at DESC
				LIMIT %d
				RETURN MERGE(
					s,
					{
						plasmid_properties: { 
							image_map: v.image_map,
							sequence: v.sequence,
							name: v.name
						} 
					}
				)	
	`
	PlasmidListFilter = `
		FOR s IN %s
			FOR v, e IN 1..1 OUTBOUND s GRAPH '%s'
				FILTER e.type == 'plasmid'
				%s
				SORT s.created_at DESC
				LIMIT %d
				RETURN MERGE(
					s,
					{
						plasmid_properties: { 
							image_map: v.image_map,
							sequence: v.sequence,
							name: v.name
						} 
					}
				)
	`
	PlasmidListWithCursor = `
		FOR s IN %s
			FOR v, e IN 1..1 OUTBOUND s GRAPH '%s'
				FILTER e.type == 'plasmid'
				FILTER s.created_at <= DATE_ISO8601(%d)
				SORT s.created_at DESC
				LIMIT %d
				RETURN MERGE(
					s,
					{
						plasmid_properties: { 
							image_map: v.image_map,
							sequence: v.sequence,
							name: v.name
						} 
					}
				)		
	`
	PlasmidListFilterWithCursor = `
		FOR s IN %s
			FOR v, e IN 1..1 OUTBOUND s GRAPH '%s'
				FILTER e.type == 'plasmid'
				%s
				FILTER s.created_at <= DATE_ISO8601(%d)
				SORT s.created_at DESC
				LIMIT %d
				RETURN MERGE(
					s,
					{
						plasmid_properties: { 
							image_map: v.image_map,
							sequence: v.sequence,
							name: v.name
						} 
					}
				)
	`
)
