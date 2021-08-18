package arangodb

import (
	"fmt"
	"strings"

	manager "github.com/dictyBase/arangomanager"
	ontoarango "github.com/dictyBase/go-obograph/storage/arangodb"
	"github.com/dictyBase/modware-stock/internal/repository"
	validator "github.com/go-playground/validator/v10"
)

type arangorepository struct {
	ontoc      *ontoarango.OntoCollection
	sess       *manager.Session
	database   *manager.Database
	stockc     *stockc
	strainOnto string
}

// NewStockRepo acts as constructor for database
func NewStockRepo(connP *manager.ConnectParams,
	collP *CollectionParams,
	ontoP *ontoarango.CollectionParams,
) (repository.StockRepository, error) {
	ar := &arangorepository{strainOnto: collP.StrainOntology}
	validate := validator.New()
	if err := validate.Struct(collP); err != nil {
		return ar, err
	}
	sess, db, err := manager.NewSessionDb(connP)
	if err != nil {
		return ar, err
	}
	oc, err := ontoarango.CreateCollection(db, ontoP)
	if err != nil {
		return ar, err
	}
	ar.ontoc = oc
	ar.sess = sess
	ar.database = db
	err = createDbStruct(ar, collP)
	return ar, err
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

func mergeBindParams(bm ...map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for _, m := range bm {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}

func genAQLDocExpression(bindVars map[string]interface{}) string {
	var bindParams []string
	for k := range bindVars {
		bindParams = append(bindParams, fmt.Sprintf("%s: @%s", k, k))
	}
	return strings.Join(bindParams, ",")
}
