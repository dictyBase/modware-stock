package arangodb

import (
	"fmt"
	"io"
	"maps"
	"slices"

	A "github.com/IBM/fp-go/array"
	F "github.com/IBM/fp-go/function"
	S "github.com/IBM/fp-go/string"
	"github.com/cockroachdb/errors"
	manager "github.com/dictyBase/arangomanager"
	ontostorage "github.com/dictyBase/go-obograph/storage"
	ontoarango "github.com/dictyBase/go-obograph/storage/arangodb"
	"github.com/dictyBase/modware-stock/internal/repository"
	"github.com/dictyBase/modware-stock/internal/repository/arangodb/statement"
	validator "github.com/go-playground/validator/v10"
)

type arangorepository struct {
	ontoc       *ontoarango.OntoCollection
	sess        *manager.Session
	database    *manager.Database
	stockc      *stockc
	strainOnto  string
	plasmidOnto string
}

// NewStockRepo acts as constructor for database
func NewStockRepo(connP *manager.ConnectParams,
	collP *CollectionParams,
	ontoP *ontoarango.CollectionParams,
) (repository.StockRepository, error) {
	ar := &arangorepository{
		strainOnto:  collP.StrainOntology,
		plasmidOnto: collP.PlasmidOntology,
	}
	validate := validator.New()
	if err := validate.Struct(collP); err != nil {
		return ar, err
	}
	sess, db, err := manager.NewSessionDb(connP)
	if err != nil {
		return ar,
			errors.Errorf("error in creating database session %s", err)
	}
	oc, err := ontoarango.CreateCollection(db, ontoP)
	if err != nil {
		return ar,
			errors.Errorf("error in creating ontology collections %s", err)
	}
	ar.ontoc = oc
	ar.sess = sess
	ar.database = db
	err = createDbStruct(ar, collP)
	return ar, err
}

func (ar *arangorepository) checkStock(id string) (string, error) {
	r, err := ar.database.GetRow(
		statement.StockFindIdQ,
		map[string]any{
			"stock_collection": ar.stockc.stock.Name(),
			"stock_prop_graph": ar.stockc.stockPropType.Name(),
			"stock_id":         id,
		})
	if err != nil {
		return id,
			errors.Errorf("error in finding stock id %s %s", id, err)
	}
	if r.IsEmpty() {
		return id,
			errors.Errorf("stock id %s is absent in database", id)
	}
	var propKey string
	if err := r.Read(&propKey); err != nil {
		return id,
			errors.Errorf("error in reading using stock id %s %s", id, err)
	}
	return propKey, nil
}

func normalizeSliceBindParam(s []string) []string {
	if len(s) > 0 {
		return s
	}
	return []string{}
}

func normalizeStrBindParam(str string) string {
	if len(str) > 0 {
		return str
	}
	return ""
}

func mergeBindParams(bm ...map[string]any) map[string]any {
	result := make(map[string]any)
	for _, m := range bm {
		maps.Copy(result, m)
	}
	return result
}

// genAQLDocExpression generates AQL document expression from bind variables
// using functional composition with fp-go and standard library iterators
func genAQLDocExpression(bindVars map[string]any) string {
	return F.Pipe4(
		bindVars,
		maps.Keys,
		slices.Collect,
		A.Map(formatAQLBinding),
		S.Join(","),
	)
}

// formatAQLBinding formats a key to AQL bind parameter format
func formatAQLBinding(key string) string {
	return fmt.Sprintf("%s: @%s", key, key)
}

func (ar *arangorepository) Dbh() *manager.Database {
	return ar.database
}

func (ar *arangorepository) termID(term, onto string) (string, error) {
	var id string
	r, err := ar.database.GetRow(
		statement.StrainExistTermQ,
		map[string]any{
			"@cv_collection":     ar.ontoc.Cv.Name(),
			"@cvterm_collection": ar.ontoc.Term.Name(),
			"ontology":           onto,
			"term":               term,
		})
	if err != nil {
		return id,
			fmt.Errorf("error in running obograph retrieving query: %w", err)
	}
	if r.IsEmpty() {
		return id,
			fmt.Errorf("ontology %s and tag %s does not exist", onto, term)
	}
	if err := r.Read(&id); err != nil {
		return id, fmt.Errorf("error in retrieving obograph id: %w", err)
	}
	return id, nil
}

func (ar *arangorepository) LoadOboJSON(
	r io.Reader,
) (*ontostorage.UploadInformation, error) {
	ds, err := ontoarango.NewDataSourceFromDb(ar.database,
		&ontoarango.CollectionParams{
			OboGraph:     ar.ontoc.Obog.Name(),
			GraphInfo:    ar.ontoc.Cv.Name(),
			Relationship: ar.ontoc.Rel.Name(),
			Term:         ar.ontoc.Term.Name(),
		})
	if err != nil {
		return &ontostorage.UploadInformation{}, err
	}
	return ontostorage.LoadOboJSONFromDataSource(r, ds)
}
