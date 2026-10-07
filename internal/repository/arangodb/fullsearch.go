package arangodb

import (
	"context"

	driver "github.com/arangodb/go-driver"
	"github.com/cockroachdb/errors"
	"github.com/dictyBase/modware-stock/internal/repository"
)

// Search asset names owned by the full search feature. Every name
// belongs to this feature alone, so other search features can be
// absent, present or stale without any effect.
const (
	// fullSearchViewName is the classic arangosearch view of this
	// feature. The AQL templates name it literally, because ArangoDB
	// does not accept a bind parameter for a view name in SEARCH.
	fullSearchViewName = "stock_full_search"
	// fullSearchNormAnalyzer lowercases a value and removes accents. It
	// serves the prefix branches.
	fullSearchNormAnalyzer = "stock_search_norm"
	// fullSearchNgramAnalyzer chains norm and ngram. It serves the fuzzy
	// branches.
	fullSearchNgramAnalyzer = "stock_search_ngram"
	// fullSearchTextAnalyzer is the built-in English text analyzer. It
	// serves the token branches and the phrase branches. It is built in,
	// so the setup must not try to create it.
	fullSearchTextAnalyzer = "text_en"
)

// fullSearchIdentifierFields lists the 8 identifier and name fields
// that the view indexes with both custom analyzers. The first three
// live on the stock collection, the last five on the stock property
// collection.
var fullSearchIdentifierFields = []struct {
	label string
	coll  fullSearchColl
}{
	{label: paramStockID, coll: fullSearchCollStock},
	{label: fieldGenes, coll: fullSearchCollStock},
	{label: fieldDbxrefs, coll: fullSearchCollStock},
	{label: fieldLabel, coll: fullSearchCollProp},
	{label: fieldNames, coll: fullSearchCollProp},
	{label: fieldSpecies, coll: fullSearchCollProp},
	{label: fieldPlasmid, coll: fullSearchCollProp},
	{label: fieldName, coll: fullSearchCollProp},
}

// fullSearchProseFields lists the 2 prose fields that the view indexes
// with the built-in text analyzer only. The plan does not index
// editable_summary: an n-gram index over long prose grows fast and
// returns weak matches.
var fullSearchProseFields = []struct {
	label string
	coll  fullSearchColl
}{
	{label: fieldSummary, coll: fullSearchCollStock},
	{label: fieldDepositor, coll: fullSearchCollStock},
}

type fullSearchColl int

const (
	fullSearchCollStock fullSearchColl = iota
	fullSearchCollProp
)

// SearchStock is a placeholder that Task 4 replaces with the real
// 19-branch query implementation.
func (ar *arangorepository) SearchStock(
	_ *repository.FullSearchQuery,
) ([]*repository.FullSearchResult, error) {
	return nil, errors.New("SearchStock is not implemented yet")
}

// ensureFullSearch creates both custom analyzers and the view. It is
// idempotent, and a failure stops the start of the service.
func (ar *arangorepository) ensureFullSearch(ctx context.Context) error {
	if _, _, err := ar.database.Handler().EnsureCreatedAnalyzer(
		ctx,
		fullSearchNormDef(),
	); err != nil {
		return errors.Errorf(
			"error in creating analyzer %s %s",
			fullSearchNormAnalyzer,
			err,
		)
	}
	if _, _, err := ar.database.Handler().EnsureCreatedAnalyzer(
		ctx,
		fullSearchNgramDef(),
	); err != nil {
		return errors.Errorf(
			"error in creating analyzer %s %s",
			fullSearchNgramAnalyzer,
			err,
		)
	}
	return ar.ensureFullSearchView(ctx)
}

// fullSearchNormDef is the norm analyzer of the prefix branches. It
// lowercases and removes accents.
func fullSearchNormDef() *driver.ArangoSearchAnalyzerDefinition {
	return &driver.ArangoSearchAnalyzerDefinition{
		Name: fullSearchNormAnalyzer,
		Type: driver.ArangoSearchAnalyzerTypeNorm,
		Properties: driver.ArangoSearchAnalyzerProperties{
			Locale: analyzerNormLocale,
			Case:   driver.ArangoSearchCaseLower,
			Accent: new(false),
		},
	}
}

// fullSearchNgramDef is the pipeline analyzer of the fuzzy branches.
// The three features are mandatory: NGRAM_MATCH fails without them.
func fullSearchNgramDef() *driver.ArangoSearchAnalyzerDefinition {
	normStep := *fullSearchNormDef()
	return &driver.ArangoSearchAnalyzerDefinition{
		Name: fullSearchNgramAnalyzer,
		Type: driver.ArangoSearchAnalyzerTypePipeline,
		Properties: driver.ArangoSearchAnalyzerProperties{
			Pipeline: []driver.ArangoSearchAnalyzerPipeline{
				{
					Type:       driver.ArangoSearchAnalyzerTypeNorm,
					Properties: normStep.Properties,
				},
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

// fullSearchLinks builds the view links from the collection names of
// the repository. It never hardcodes a production name. The identifier
// fields carry both custom analyzers; the prose fields carry the
// built-in text analyzer only, and no field carries both.
func (ar *arangorepository) fullSearchLinks() driver.ArangoSearchLinks {
	stockFields := driver.ArangoSearchFields{}
	propFields := driver.ArangoSearchFields{}
	identifier := []string{fullSearchNormAnalyzer, fullSearchNgramAnalyzer}
	prose := []string{fullSearchTextAnalyzer}
	for _, f := range fullSearchIdentifierFields {
		if f.coll == fullSearchCollStock {
			stockFields[f.label] = driver.ArangoSearchElementProperties{
				Analyzers: identifier,
			}
		} else {
			propFields[f.label] = driver.ArangoSearchElementProperties{
				Analyzers: identifier,
			}
		}
	}
	for _, f := range fullSearchProseFields {
		if f.coll == fullSearchCollStock {
			stockFields[f.label] = driver.ArangoSearchElementProperties{
				Analyzers: prose,
			}
		} else {
			propFields[f.label] = driver.ArangoSearchElementProperties{
				Analyzers: prose,
			}
		}
	}
	return driver.ArangoSearchLinks{
		ar.stockc.stock.Name():     {Fields: stockFields},
		ar.stockc.stockProp.Name(): {Fields: propFields},
	}
}

// ensureFullSearchView creates the view or reconciles a stale one.
// A stale view converges through SetProperties, which is a full
// replacement; the view itself is never deleted.
func (ar *arangorepository) ensureFullSearchView(ctx context.Context) error {
	db := ar.database.Handler()
	exists, err := db.ViewExists(ctx, fullSearchViewName)
	if err != nil {
		return errors.Errorf(
			"error in checking view %s %s",
			fullSearchViewName,
			err,
		)
	}
	if !exists {
		_, err := db.CreateArangoSearchView(
			ctx,
			fullSearchViewName,
			&driver.ArangoSearchViewProperties{Links: ar.fullSearchLinks()},
		)
		// A second repository instance can win the race.
		if err != nil && !driver.IsConflict(err) {
			return errors.Errorf(
				"error in creating view %s %s",
				fullSearchViewName,
				err,
			)
		}
		return nil
	}
	v, err := db.View(ctx, fullSearchViewName)
	if err != nil {
		return errors.Errorf("error in opening view %s %s", fullSearchViewName, err)
	}
	asv, err := v.ArangoSearchView()
	if err != nil {
		// A view of another type holds the name. Do not delete it.
		return errors.Errorf(
			"view %s is not an arangosearch view %s",
			fullSearchViewName,
			err,
		)
	}
	props, err := asv.Properties(ctx)
	if err != nil {
		return errors.Errorf(
			"error in reading view %s properties %s",
			fullSearchViewName,
			err,
		)
	}
	want := ar.fullSearchLinks()
	if sameLinkShape(props.Links, want) {
		return nil
	}
	if err := asv.SetProperties(
		ctx,
		driver.ArangoSearchViewProperties{Links: want},
	); err != nil {
		return errors.Errorf(
			"error in reconciling view %s %s",
			fullSearchViewName,
			err,
		)
	}
	return nil
}
