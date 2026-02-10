// Package nats provides NATS messaging implementation for publishing stock events.
package nats

import (
	"fmt"

	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/message"
	gnats "github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

type natsPublisher struct {
	conn *gnats.Conn
}

// NewPublisher creates a new NATS publisher connected to the specified host and port.
func NewPublisher(
	host, port string,
	options ...gnats.Option,
) (message.Publisher, error) {
	nc, err := gnats.Connect(
		fmt.Sprintf("nats://%s:%s", host, port),
		options...)
	if err != nil {
		return &natsPublisher{}, err
	}
	return &natsPublisher{conn: nc}, nil
}

func (n *natsPublisher) PublishStrain(subj string, s *stock.Strain) error {
	data, err := proto.Marshal(s)
	if err != nil {
		return fmt.Errorf("failed to marshal strain: %w", err)
	}
	return n.conn.Publish(subj, data)
}

func (n *natsPublisher) PublishPlasmid(subj string, p *stock.Plasmid) error {
	data, err := proto.Marshal(p)
	if err != nil {
		return fmt.Errorf("failed to marshal plasmid: %w", err)
	}
	return n.conn.Publish(subj, data)
}

func (n *natsPublisher) Close() error {
	n.conn.Close()
	return nil
}
