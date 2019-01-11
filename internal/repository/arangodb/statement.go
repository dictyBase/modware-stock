package arangodb

const (
	stockFindIdQ = `
		FOR s IN @@stock_collection
			FOR v IN 1..1 OUTBOUND s GRAPH @graph
				FILTER s.stock_id == @id
				LIMIT 1
				RETURN { key: s._key, propkey: v._key }
	`
	stockFindQ = `
		FOR s IN @@stock_collection
			FILTER s.stock_id == @id
			LIMIT 1
			RETURN s._id
	`
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
				label: @label,
				species: @species,
				plasmid: @plasmid,
				names: @names
			} INTO @@stock_properties_collection RETURN NEW
		)
		INSERT { _from: n[0]._id, _to: o[0]._id, type: 'strain' } INTO @@stock_type_collection
		RETURN MERGE(
			n[0],
			{
				strain_properties: o[0]
			}
		)
	`
	stockStrainWithParentsIns = `
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
				label: @label,
				species: @species,
				plasmid: @plasmid,
				names: @names
			} INTO @@stock_properties_collection RETURN NEW
		)
		INSERT { _from: n[0]._id, _to: o[0]._id, type: 'strain' } INTO @@stock_type_collection
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
				stock_id: CONCAT("DBP0", kg[0]._key)
			} INTO @@stock_collection RETURN NEW
		)
		LET o = (
			INSERT {
				image_map: @image_map,
				sequence: @sequence
			} INTO @@stock_properties_collection RETURN NEW
		)
		INSERT { _from: n[0]._id, _to: o[0]._id, type: 'plasmid' } INTO @@stock_type_collection
		RETURN MERGE(
			n[0],
			{
				plasmid_properties: o[0]
			}
		)
	`
	stockGetStrain = `
		FOR s IN @@stock_collection
			FOR v, e IN 1..1 OUTBOUND s GRAPH @graph
				FILTER e.type == 'strain'
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
			FOR v, e IN 1..1 OUTBOUND s GRAPH @graph
				FILTER e.type == 'strain'
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
	strainUpd = `
		LET s = (
			UPDATE { _key: @key } WITH { %s }
			IN @@stock_collection RETURN NEW
		)
		LET p = (
			UPDATE { _key: @propkey } WITH { %s }
			IN @@stock_properties_collection
			RETURN {
				strain_properties: {
					systematic_name: NEW.systematic_name,
					label: NEW.label,
					species: NEW.species,
					plasmid: NEW.plasmid,
					names: NEW.names
				}
			}
		)
		RETURN MERGE(s[0],p[0])
	`
	strainWithParentUpd = `
		LET s = (
			UPDATE { _key: @key } WITH { %s }
			IN @@stock_collection RETURN NEW
		)
		LET p = (
			UPDATE { _key: @propkey } WITH { %s }
			IN @@stock_properties_collection
			RETURN {
				strain_properties: {
					systematic_name: NEW.systematic_name,
					label: NEW.label,
					species: NEW.species,
					plasmid: NEW.plasmid,
					names: NEW.names
				}
			}
		)
		FOR p in @parents
			FOR v,e IN 1..1 OUTBOUND p GRAPH @parent_graph
				REMOVE e IN @@stock_collection
		FOR p in @parents
			INSERT { _from: p, _to: s[0]._id } INTO @@parent_strain_collection
		RETURN MERGE(s[0],p[0])
	`
	plasmidUpd = `
		LET s = (
			UPDATE { _key: @key } WITH { %s }
			IN @@stock_collection RETURN NEW
		)
		LET p = (
			UPDATE { _key: @propkey } WITH { %s }
			IN @@stock_properties_collection
			RETURN {
				plasmid_properties: {
					image_map: NEW.image_map,
					sequence: NEW.sequence
				}
			}
		)
		RETURN MERGE(s[0],p[0])
	`
	parentUpd = `
		UPDATE { _key: @key }
			WITH { _to: @parent }
			IN @@parent_strain_collection RETURN NEW
	`
	stockList = `
		FOR s IN %s
			SORT s.created_at DESC
			LIMIT %d
			RETURN s
	`
	stockListFilter = `
		FOR s in %s
			%s
			SORT s.created_at DESC
			LIMIT %d
			RETURN s
	`
	stockListWithCursor = `
		FOR s in %s
			FILTER s.created_at <= DATE_ISO8601(%d)
			SORT s.created_at DESC
			LIMIT %d
			RETURN s
	`
	stockListFilterWithCursor = `
		FOR s in %s
			FILTER s.created_at <= DATE_ISO8601(%d)
			%s
			SORT s.created_at DESC
			LIMIT %d
			RETURN s	
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
							label: v.label,
							species: v.species,
							plasmid: v.plasmid,
							parents: v.parents,
							names: v.names
						}
					}
				)
	`
	strainListFilter = `
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
							label: v.label,
							species: v.species,
							plasmid: v.plasmid,
							parents: v.parents,
							names: v.names
						}
					}
				)
	`
	strainListFilterWithCursor = `
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
	plasmidListFilter = `
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
	plasmidListFilterWithCursor = `
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
								sequence: v.sequence
							} 
						}
					)
	`
	stockDelQ = `
		REMOVE @key IN @@stock_collection
	`
)
