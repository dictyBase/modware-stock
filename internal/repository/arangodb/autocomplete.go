package arangodb

import (
	"context"
	"slices"

	driver "github.com/arangodb/go-driver"
	"github.com/cockroachdb/errors"
	"github.com/dictyBase/modware-stock/internal/repository"
)

// Search asset names owned by the autocomplete feature. Every name
// belongs to this feature alone, so other search features can be
// absent, present or stale without any effect.
const (
	// autocompleteViewName is the classic arangosearch view of this
	// feature. The AQL templates name it literally, because ArangoDB
	// does not accept a bind parameter for a view name in SEARCH.
	autocompleteViewName = "stock_autocomplete"
	// autocompleteNormAnalyzer lowercases a value and removes accents.
	// It serves the prefix branches.
	autocompleteNormAnalyzer = "stock_autocomplete_norm"
	// autocompleteNgramAnalyzer chains norm and ngram. It serves the
	// fuzzy branches.
	autocompleteNgramAnalyzer = "stock_autocomplete_ngram"
)

// autocompleteStockFields lists the 8 indexed fields in the order of the
// branch matrix. The first three run over the stock collection, the last
// five over the stock property collection.
var autocompleteStockFields = []autocompleteField{
	{label: paramStockID, coll: autocompleteCollStock, display: autocompleteScalar},
	{label: fieldGenes, coll: autocompleteCollStock, display: autocompleteArray},
	{label: fieldDbxrefs, coll: autocompleteCollStock, display: autocompleteArray},
	{label: fieldLabel, coll: autocompleteCollProp, display: autocompleteScalar},
	{label: fieldNames, coll: autocompleteCollProp, display: autocompleteArray},
	{label: fieldSpecies, coll: autocompleteCollProp, display: autocompleteScalar},
	{label: fieldPlasmid, coll: autocompleteCollProp, display: autocompleteScalar},
	{label: fieldName, coll: autocompleteCollProp, display: autocompleteScalar},
}

type autocompleteColl int

const (
	autocompleteCollStock autocompleteColl = iota
	autocompleteCollProp
)

type autocompleteDisplay int

const (
	autocompleteScalar autocompleteDisplay = iota
	autocompleteArray
)

type autocompleteField struct {
	label   string
	coll    autocompleteColl
	display autocompleteDisplay
}

// ensureAutocompleteSearch creates both analyzers and the view. It is
// idempotent, and a failure stops the start of the service.
func (ar *arangorepository) ensureAutocompleteSearch(ctx context.Context) error {
	_, _, err := ar.database.Handler().EnsureCreatedAnalyzer(
		ctx,
		autocompleteNormDef(),
	)
	if err != nil {
		return errors.Errorf(
			"error in creating analyzer %s %s",
			autocompleteNormAnalyzer,
			err,
		)
	}
	_, _, err = ar.database.Handler().EnsureCreatedAnalyzer(
		ctx,
		autocompleteNgramDef(),
	)
	if err != nil {
		return errors.Errorf(
			"error in creating analyzer %s %s",
			autocompleteNgramAnalyzer,
			err,
		)
	}
	return ar.ensureAutocompleteView(ctx)
}

// autocompleteNormDef is the norm analyzer of the prefix branches. It
// lowercases and removes accents.
func autocompleteNormDef() *driver.ArangoSearchAnalyzerDefinition {
	return &driver.ArangoSearchAnalyzerDefinition{
		Name: autocompleteNormAnalyzer,
		Type: driver.ArangoSearchAnalyzerTypeNorm,
		Properties: driver.ArangoSearchAnalyzerProperties{
			Locale: "en.utf-8",
			Case:   driver.ArangoSearchCaseLower,
			Accent: new(false),
		},
	}
}

// autocompleteNgramDef is the pipeline analyzer of the fuzzy branches.
// The three features are mandatory: NGRAM_MATCH fails without them.
func autocompleteNgramDef() *driver.ArangoSearchAnalyzerDefinition {
	normStep := *autocompleteNormDef()
	return &driver.ArangoSearchAnalyzerDefinition{
		Name: autocompleteNgramAnalyzer,
		Type: driver.ArangoSearchAnalyzerTypePipeline,
		Properties: driver.ArangoSearchAnalyzerProperties{
			Pipeline: []driver.ArangoSearchAnalyzerPipeline{
				{Type: driver.ArangoSearchAnalyzerTypeNorm, Properties: normStep.Properties},
				{
					Type: driver.ArangoSearchAnalyzerTypeNGram,
					Properties: driver.ArangoSearchAnalyzerProperties{
						Min:              new(int64(2)),
						Max:              new(int64(3)),
						PreserveOriginal: new(true),
						StreamType:       new(driver.ArangoSearchNGramStreamUTF8),
					},
				},
			},
		},
		Features: []driver.ArangoSearchAnalyzerFeature{
			driver.ArangoSearchAnalyzerFeatureFrequency,
			driver.ArangoSearchAnalyzerFeatureNorm,
			driver.ArangoSearchAnalyzerFeaturePosition,
		},
	}
}

// autocompleteLinks builds the view links from the collection names of
// the repository. It never hardcodes a production name.
func (ar *arangorepository) autocompleteLinks() driver.ArangoSearchLinks {
	stockFields := driver.ArangoSearchFields{}
	propFields := driver.ArangoSearchFields{}
	analyzers := []string{autocompleteNormAnalyzer, autocompleteNgramAnalyzer}
	for _, f := range autocompleteStockFields {
		if f.coll == autocompleteCollStock {
			stockFields[f.label] = driver.ArangoSearchElementProperties{
				Analyzers: analyzers,
			}
		} else {
			propFields[f.label] = driver.ArangoSearchElementProperties{
				Analyzers: analyzers,
			}
		}
	}
	return driver.ArangoSearchLinks{
		ar.stockc.stock.Name():     {Fields: stockFields},
		ar.stockc.stockProp.Name(): {Fields: propFields},
	}
}

// ensureAutocompleteView creates the view or reconciles a stale one.
// A stale view converges through SetProperties, which is a full
// replacement; the view itself is never deleted.
func (ar *arangorepository) ensureAutocompleteView(ctx context.Context) error {
	db := ar.database.Handler()
	exists, err := db.ViewExists(ctx, autocompleteViewName)
	if err != nil {
		return errors.Errorf(
			"error in checking view %s %s",
			autocompleteViewName,
			err,
		)
	}
	if !exists {
		_, err := db.CreateArangoSearchView(
			ctx,
			autocompleteViewName,
			&driver.ArangoSearchViewProperties{Links: ar.autocompleteLinks()},
		)
		// A second repository instance can win the race.
		if err != nil && !driver.IsConflict(err) {
			return errors.Errorf(
				"error in creating view %s %s",
				autocompleteViewName,
				err,
			)
		}
		return nil
	}
	v, err := db.View(ctx, autocompleteViewName)
	if err != nil {
		return errors.Errorf("error in opening view %s %s", autocompleteViewName, err)
	}
	asv, err := v.ArangoSearchView()
	if err != nil {
		// A view of another type holds the name. Do not delete it.
		return errors.Errorf(
			"view %s is not an arangosearch view %s",
			autocompleteViewName,
			err,
		)
	}
	props, err := asv.Properties(ctx)
	if err != nil {
		return errors.Errorf(
			"error in reading view %s properties %s",
			autocompleteViewName,
			err,
		)
	}
	want := ar.autocompleteLinks()
	if sameLinkShape(props.Links, want) {
		return nil
	}
	if err := asv.SetProperties(ctx, driver.ArangoSearchViewProperties{Links: want}); err != nil {
		return errors.Errorf(
			"error in reconciling view %s %s",
			autocompleteViewName,
			err,
		)
	}
	return nil
}

// sameLinkShape compares only the shape that this feature controls: the
// link keys, the field names per link and the analyzer names per field.
// A deep comparison would always report a difference, because the
// server fills defaults into the response.
func sameLinkShape(got, want driver.ArangoSearchLinks) bool {
	if len(got) != len(want) {
		return false
	}
	for coll, gl := range got {
		wl, ok := want[coll]
		if !ok || !sameFieldShape(gl.Fields, wl.Fields) {
			return false
		}
	}
	return true
}

func sameFieldShape(got, want driver.ArangoSearchFields) bool {
	if len(got) != len(want) {
		return false
	}
	for field, gf := range got {
		wf, ok := want[field]
		if !ok || !slices.Equal(gf.Analyzers, wf.Analyzers) {
			return false
		}
	}
	return true
}

// AutocompleteStock returns suggestions for a partial stock identifier,
// name or attribute value. The full query builder lands with the
// autocomplete search implementation.
func (ar *arangorepository) AutocompleteStock(
	params *repository.AutocompleteQuery,
) ([]*repository.Suggestion, error) {
	if params == nil {
		return nil, errors.New("expect a non-nil autocomplete query")
	}
	return nil, errors.New("autocomplete search is not implemented yet")
}
