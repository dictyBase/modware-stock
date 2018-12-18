package message

import (
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
)

// Publisher manages publishing of message
type Publisher interface {
	// Publish publishes the stock object using the given subject
	Publish(subject string, st *stock.Stock) error
	// Close closes the connection to the underlying messaging server
	Close() error
}
