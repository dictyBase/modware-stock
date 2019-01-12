package statement

const (
	StockUpd = `
		UPDATE { _key: @key }
			WITH { %s }
			IN @@stock_collection RETURN NEW
	`
	StrainUpd = `
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
	StrainWithParentUpd = `
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
			FOR v,e IN 1..1 INBOUND s[0] GRAPH @parent_graph
				UPDATE e WITH { _from: p } IN @parent_strain_collection
		RETURN MERGE(s[0],p[0])
	`
	PlasmidUpd = `
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
)
