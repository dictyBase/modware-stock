package arangodb

const (
	stockIns = `
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
			strain_properties: {
				systematic_name: @systematic_name,
				descriptor: @descriptor,
				species: @species,
				plasmid: @plasmid,
				parents: @parents,
				names: @names
			},
			plasmid_properties: {
				image_map: @image_map,
				sequence: @sequence,
				keywords: @keywords
			}
		} INTO @@stocks_collection RETURN NEW
	`
	stockGet = `
		FOR stock IN @@stocks_collection
			FILTER stock._key == @key
			RETURN stock
	`
	stockUpd = `
		UPDATE { _key: @key }
			WITH { %s }
			IN @@stocks_collection RETURN NEW
	`
	stockList = `
		FOR stock IN @@stocks_collection
			SORT stock.created_at DESC
			LIMIT @limit
			RETURN stock
	`
	stockListWithCursor = `
		FOR stock in @@stocks_collection
			FILTER stock.created_at <= DATE_ISO8601(@next_cursor)
			SORT stock.created_at DESC
			LIMIT @limit
			RETURN stock
	`
	stockDelQ = `
		REMOVE @key IN @@stocks_collection
	`
)
