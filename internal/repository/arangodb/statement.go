package arangodb

const (
	stockStrainIns = `
		LET kg = (
			INSERT {} INTO @@stock_key_generator RETURN NEW
		)
		LET n = (
			INSERT {
				created_at: DATE_ISO8601(DATE_NOW()),
				updated_at: DATE_ISO8601(DATE_NOW()),
				created_by: @created_by,
				updated_by: @updated_by,
				summary: @summary,
				editable_summary: @editable_summary,
				depositor: @depositor,
				genes: @genes,
				dbxrefs: @dbxrefs,
				publications: @publications,
				stock_id: CONCAT("DBS0", kg[0]._key)
			} INTO @@stock_collection RETURN NEW
		)
		LET o = (
			INSERT {
				systematic_name: @systematic_name,
				descriptor: @descriptor,
				species: @species,
				plasmid: @plasmid,
				parents: @parents,
				names: @names
			} INTO @@strain_collection RETURN NEW
		)
		INSERT { _from: n[0]._id, _to: o[0]._id } INTO @@stock_strain_collection
		FOR p IN @parents
			INSERT { _from: p, _to: n[0]._id } INTO @@parent_strain_collection
		RETURN MERGE(
			n[0],
			{
				strain_properties: o[0]
			}
		)
	`
	stockPlasmidIns = `
		LET kg = (
			INSERT {} INTO @@stock_key_generator RETURN NEW
		)
		LET n = (
			INSERT {
				created_at: DATE_ISO8601(DATE_NOW()),
				updated_at: DATE_ISO8601(DATE_NOW()),
				created_by: @created_by,
				updated_by: @updated_by,
				summary: @summary,
				editable_summary: @editable_summary,
				depositor: @depositor,
				genes: @genes,
				dbxrefs: @dbxrefs,
				publications: @publications,
				stock_id: CONCAT("DBS0", kg[0]._key)
			} INTO @@stock_collection RETURN NEW
		)
		LET o = (
			INSERT {
				image_map: @image_map,
				sequence: @sequence
			} INTO @@plasmid_collection RETURN NEW
		)
		INSERT { _from: n[0]._id, _to: o[0]._id } INTO @@stock_plasmid_collection
		RETURN MERGE(
			n[0],
			{
				plasmid_properties: o[0]
			}
		)
	`
	stockGetStrain = `
		FOR s IN @@stock_collection
			FOR v IN 1..1 OUTBOUND s GRAPH @graph
				FILTER s.stock_id == @id
				RETURN MERGE(
					s,
					{
						strain_properties: { 
							systematic_name: v.systematic_name, 
							descriptor: v.descriptor, 
							species: v.species, 
							plasmid: v.plasmid, 
							parents: v.parents, 
							names: v.names
						} 
					}
				)
	`
	stockGetPlasmid = `
		FOR s IN @@stock_collection
			FOR v IN 1..1 OUTBOUND s GRAPH @graph
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
	stockUpd = `
		UPDATE { _key: @key }
			WITH { %s }
			IN @@stock_collection RETURN NEW
	`
	parentUpd = `
		UPDATE { _key: @key }
			WITH { _to: @parent }
			IN @@parent_strain_collection RETURN NEW
	`
	stockList = `
		FOR stock IN @@stock_collection
			SORT stock.created_at DESC
			LIMIT @limit
			RETURN stock
	`
	stockListWithCursor = `
		FOR stock in @@stock_collection
			FILTER stock.created_at <= DATE_ISO8601(@next_cursor)
			SORT stock.created_at DESC
			LIMIT @limit
			RETURN stock
	`
	strainList = `
		FOR s IN @@stock_collection
			FOR v IN 1..1 OUTBOUND s GRAPH @graph
				SORT s.created_at DESC
				LIMIT @limit
				RETURN MERGE(
					s,
					{
						strain_properties: { 
							systematic_name: v.systematic_name, 
							descriptor: v.descriptor, 
							species: v.species, 
							plasmid: v.plasmid, 
							parents: v.parents, 
							names: v.names
						} 
					}
				)
	`
	strainListWithCursor = `
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
							descriptor: v.descriptor, 
							species: v.species, 
							plasmid: v.plasmid, 
							parents: v.parents, 
							names: v.names
						} 
					}
				)
	`
	plasmidList = `
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
	plasmidListWithCursor = `
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
	stockDelQ = `
		REMOVE @key IN @@stock_collection
	`
)
