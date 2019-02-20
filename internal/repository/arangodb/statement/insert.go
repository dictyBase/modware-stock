package statement

const (
	StockStrainIns = `
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
				stock_id: CONCAT("DBS0", kg[0]._key),
				_key: CONCAT("DBS0", kg[0]._key)
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
	StockStrainWithParentsIns = `
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
				stock_id: CONCAT("DBS0", kg[0]._key),
				_key: CONCAT("DBS0", kg[0]._key)
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
		INSERT { _from: @pid, _to: n[0]._id } INTO @@parent_strain_collection
		RETURN MERGE(
			n[0],
			{
				strain_properties: o[0]
			}
		)
	`
	StockStrainLoad = `
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
				stock_id: @stock_id,
				_key: @stock_id
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
	StockStrainWithParentLoad = `
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
				stock_id: @stock_id,
				_key: @stock_id
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
		INSERT { _from: @pid, _to: n[0]._id } INTO @@parent_strain_collection
		RETURN MERGE(
			n[0],
			{
				strain_properties: o[0]
			}
		)
	`
	StockPlasmidIns = `
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
				stock_id: CONCAT("DBP0", kg[0]._key),
				_key: CONCAT("DBP0", kg[0]._key)
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
	StockPlasmidLoad = `
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
				stock_id: @stock_id,
				_key: @stock_id
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
)
