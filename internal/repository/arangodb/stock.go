package arangodb

import (
	"context"

	driver "github.com/arangodb/go-driver"
	"github.com/cockroachdb/errors"
)

// RemoveStock removes a stock
func (ar *arangorepository) RemoveStock(id string) error {
	found, err := ar.stockc.stock.DocumentExists(context.Background(), id)
	if err != nil {
		return errors.Errorf("error in finding document with id %s %s", id, err)
	}
	if !found {
		return errors.Errorf("document not found %s", id)
	}
	_, err = ar.stockc.stock.RemoveDocument(
		driver.WithSilent(context.Background()),
		id,
	)
	if err != nil {
		return errors.Errorf("error in removing document with id %s %s", id, err)
	}
	return nil
}
