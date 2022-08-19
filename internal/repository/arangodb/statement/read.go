package statement

const (
	StockFindIdQ = `
		FOR stock_prop IN 1..1 OUTBOUND
			CONCAT(@stock_collection,"/",@stock_id) GRAPH @stock_prop_graph
			RETURN stock_prop._key
	`
	StockFindQ = `
		FOR s IN @@stock_collection
			FILTER s.stock_id == @id
			LIMIT 1
			RETURN s._id
	`
	StockGetStrain = `
		FOR stock_prop, e IN 1..1 OUTBOUND CONCAT(@stock_collection,"/",@id) GRAPH @stock_prop_graph
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
							label: stock_prop.label,
							species: stock_prop.species,
							plasmid: stock_prop.plasmid,
							names: stock_prop.names,
							dicty_strain_property: term[0],
							parent: parent[0]
			}})
	`
	StockGetPlasmid = `
		FOR s IN @@stock_collection
			FOR stock_prop, e IN 1..1 OUTBOUND s GRAPH @stock_prop_graph
				FILTER e.type == 'plasmid'
				FILTER s.stock_id == @id
				RETURN MERGE(
					s,
					{
						plasmid_properties: {
							image_map: stock_prop.image_map,
							sequence: stock_prop.sequence,
							name: stock_prop.name
						}
					}
				)
	`
	StrainGetParentRel = `
		FOR stock_prop,e IN 1..1 INBOUND @strain_key GRAPH @parent_graph
			RETURN e._key
	`
	StrainListFromIds = `
		FOR id IN @ids
			FOR stock_prop, e IN 1..1 OUTBOUND CONCAT(@stock_collection,"/",id) GRAPH @stock_prop_graph
				LET parent = (
					FOR p IN 1..1 INBOUND CONCAT(@stock_collection,"/",id) GRAPH @parent_graph
						RETURN p.stock_id
				)
				LET term = (
					FOR cg IN 1..1 OUTBOUND CONCAT(@stock_collection,"/",id) GRAPH @stock_cvterm_graph
						FOR cv IN @@cv_collection
							FILTER cg.deprecated == false
							FILTER cg.graph_id == cv._id
							FILTER cv.metadata.namespace == @ontology
							RETURN cg.label
				)
				FILTER e.type == 'strain'
				FOR s IN @@stock_collection
					FILTER s.stock_id == id
					LIMIT @limit
					RETURN MERGE(s,{
							strain_properties: {
								parent: parent[0],
								dicty_strain_property: term[0],
								label: stock_prop.label,
								species: stock_prop.species,
								plasmid: stock_prop.plasmid,
								names: stock_prop.names
						}
					})
	`
	StrainListFilter = `
		FOR cvterm in @@cvterm_collection
			FOR cv IN @@cv_collection
				FOR s IN 1..1 INBOUND cvterm GRAPH @stock_cvterm_graph
					FOR stock_prop,etype IN 1..1 OUTBOUND s GRAPH @stock_prop_graph
						FILTER cvterm.graph_id == cv._id
						FILTER etype.type == 'strain'
						%s
						SORT s.created_at DESC
						LIMIT @limit
						RETURN MERGE(s,{
								strain_properties: { 
									label: stock_prop.label, 
									species: stock_prop.species, 
									plasmid: stock_prop.plasmid, 
									names: stock_prop.names
								} 
							}
						)
	`
	StrainListFilterWithCursor = `
		FOR cvterm in @@cvterm_collection
			FOR cv IN @@cv_collection
				FOR s IN 1..1 INBOUND cvterm GRAPH @stock_cvterm_graph
					FOR stock_prop,etype IN 1..1 OUTBOUND s GRAPH @stock_prop_graph
						FILTER cvterm.graph_id == cv._id
						FILTER etype.type == 'strain'
						%s
						FILTER s.created_at <= DATE_ISO8601(@cursor)
						SORT s.created_at DESC
						LIMIT @limit
						RETURN MERGE(s,{
								strain_properties: { 
									label: stock_prop.label, 
									species: stock_prop.species, 
									plasmid: stock_prop.plasmid, 
									names: stock_prop.names
								} 
							}
						)
	`
	StrainList = `
		FOR s IN @@stock_collection
			FOR stock_prop, e IN 1..1 OUTBOUND s GRAPH @stock_prop_graph
				FILTER e.type == 'strain'
				SORT s.created_at DESC
				LIMIT @limit
				RETURN MERGE(
					s,
					{
						strain_properties: { 
							label: stock_prop.label, 
							species: stock_prop.species, 
							plasmid: stock_prop.plasmid, 
							names: stock_prop.names
						} 
					}
				)
	`
	StrainListWithCursor = `
		FOR s IN @@stock_collection
			FOR stock_prop, e IN 1..1 OUTBOUND s GRAPH @stock_prop_graph
				FILTER e.type == 'strain'
				FILTER s.created_at <= DATE_ISO8601(@cursor)
				SORT s.created_at DESC
				LIMIT @limit
				RETURN MERGE(
					s,
					{
						strain_properties: { 
							label: stock_prop.label, 
							species: stock_prop.species, 
							plasmid: stock_prop.plasmid, 
							names: stock_prop.names
						} 
					}
				)
	`
	PlasmidList = `
		FOR s IN %s
			FOR stock_prop, e IN 1..1 OUTBOUND s GRAPH '%s'
				FILTER e.type == 'plasmid'
				SORT s.created_at DESC
				LIMIT %d
				RETURN MERGE(
					s,
					{
						plasmid_properties: { 
							image_map: stock_prop.image_map,
							sequence: stock_prop.sequence,
							name: stock_prop.name
						} 
					}
				)	
	`
	PlasmidListFilter = `
		FOR s IN %s
			FOR stock_prop, e IN 1..1 OUTBOUND s GRAPH '%s'
				FILTER e.type == 'plasmid'
				%s
				SORT s.created_at DESC
				LIMIT %d
				RETURN MERGE(
					s,
					{
						plasmid_properties: { 
							image_map: stock_prop.image_map,
							sequence: stock_prop.sequence,
							name: stock_prop.name
						} 
					}
				)
	`
	PlasmidListWithCursor = `
		FOR s IN %s
			FOR stock_prop, e IN 1..1 OUTBOUND s GRAPH '%s'
				FILTER e.type == 'plasmid'
				FILTER s.created_at <= DATE_ISO8601(%d)
				SORT s.created_at DESC
				LIMIT %d
				RETURN MERGE(
					s,
					{
						plasmid_properties: { 
							image_map: stock_prop.image_map,
							sequence: stock_prop.sequence,
							name: stock_prop.name
						} 
					}
				)		
	`
	PlasmidListFilterWithCursor = `
		FOR s IN %s
			FOR stock_prop, e IN 1..1 OUTBOUND s GRAPH '%s'
				FILTER e.type == 'plasmid'
				%s
				FILTER s.created_at <= DATE_ISO8601(%d)
				SORT s.created_at DESC
				LIMIT %d
				RETURN MERGE(
					s,
					{
						plasmid_properties: { 
							image_map: stock_prop.image_map,
							sequence: stock_prop.sequence,
							name: stock_prop.name
						} 
					}
				)
	`
)
