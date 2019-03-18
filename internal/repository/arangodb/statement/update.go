package statement

const (
	StockUpd = `
		UPDATE { _key: @key }
			WITH { updated_at: DATE_ISO8601(DATE_NOW()), %s }
			IN @@stock_collection RETURN NEW
	`
	StrainUpd = `
		LET s = (
			UPDATE { _key: @key } WITH { updated_at: DATE_ISO8601(DATE_NOW()), %s }
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
	StrainWithNewParentUpd = `
		LET s = (
			UPDATE { _key: @key } WITH { updated_at: DATE_ISO8601(DATE_NOW()), %s }
			IN @@stock_collection RETURN NEW
		)
		LET prop = (
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
		INSERT {
			_from: CONCAT(@stock_collection,'/',@parent),
			_to: CONCAT(@stock_collection,'/',@key)
		} INTO @@parent_strain_collection

		RETURN MERGE(s[0],prop[0])
	`
	StrainWithExistingParentUpd = `
		LET s = (
			UPDATE { _key: @key } WITH { updated_at: DATE_ISO8601(DATE_NOW()), %s }
			IN @@stock_collection RETURN NEW
		)
		LET prop = (
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
		UPDATE @pkey
			WITH { _from: CONCAT(@stock_collection,'/',@parent) }
			IN @@parent_strain_collection

		RETURN MERGE(s[0],prop[0])
	`
	PlasmidUpd = `
		LET s = (
			UPDATE { _key: @key } WITH { updated_at: DATE_ISO8601(DATE_NOW()), %s }
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
