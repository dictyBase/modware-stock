package nats

import (
	"fmt"

	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/message"
	"github.com/nats-io/nats.go/encoders/protobuf"
	gnats "github.com/nats-io/nats.go"
)

type natsPublisher struct {
	econn *gnats.EncodedConn
}

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
	ec, err := gnats.NewEncodedConn(nc, protobuf.PROTOBUF_ENCODER)
	if err != nil {
		return &natsPublisher{}, err
	}
	return &natsPublisher{econn: ec}, nil
}

func (n *natsPublisher) PublishStrain(subj string, s *stock.Strain) error {
	return n.econn.Publish(subj, s)
}

func (n *natsPublisher) PublishPlasmid(subj string, p *stock.Plasmid) error {
	return n.econn.Publish(subj, p)
}

func (n *natsPublisher) Close() error {
	n.econn.Close()
	return nil
}
