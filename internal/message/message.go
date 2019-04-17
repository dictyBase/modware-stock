package message

import (
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
)

// Publisher manages publishing of message
type Publisher interface {
	// PublishStrain publishes the strain object using the given subject
	PublishStrain(subject string, s *stock.Strain) error
	// PublishPlasmid publishes the plasmid object using the given subject
	PublishPlasmid(subject string, p *stock.Plasmid) error
	// Close closes the connection to the underlying messaging server
	Close() error
}
