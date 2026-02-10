package nats

import (
	"context"
	"testing"
	"time"

	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	gnats "github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// verifyReceivedStrain waits for and verifies a strain message
func verifyReceivedStrain(ctx context.Context, t *testing.T, msgChan <-chan *gnats.Msg, expected *stock.Strain) {
	t.Helper()
	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	select {
	case msg := <-msgChan:
		require.NotNil(t, msg, "Received nil message")
		require.NotEmpty(t, msg.Data, "Received empty message data")
		receivedStrain := &stock.Strain{}
		err := proto.Unmarshal(msg.Data, receivedStrain)
		require.NoError(t, err, "Failed to unmarshal received strain")
		require.Equal(t, expected.Data.Id, receivedStrain.Data.Id)
		require.Equal(t, expected.Data.Type, receivedStrain.Data.Type)
		require.Equal(t, expected.Data.Attributes.Label, receivedStrain.Data.Attributes.Label)
		require.Equal(t, expected.Data.Attributes.Species, receivedStrain.Data.Attributes.Species)
		require.Equal(t, expected.Data.Attributes.CreatedBy, receivedStrain.Data.Attributes.CreatedBy)
		require.ElementsMatch(t, expected.Data.Attributes.Genes, receivedStrain.Data.Attributes.Genes)
	case <-timeoutCtx.Done():
		t.Fatal("Timeout waiting for published message")
	}
}

// verifyReceivedPlasmid waits for and verifies a plasmid message
func verifyReceivedPlasmid(ctx context.Context, t *testing.T, msgChan <-chan *gnats.Msg, expected *stock.Plasmid) {
	t.Helper()
	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	select {
	case msg := <-msgChan:
		require.NotNil(t, msg, "Received nil message")
		require.NotEmpty(t, msg.Data, "Received empty message data")
		receivedPlasmid := &stock.Plasmid{}
		err := proto.Unmarshal(msg.Data, receivedPlasmid)
		require.NoError(t, err, "Failed to unmarshal received plasmid")
		require.Equal(t, expected.Data.Id, receivedPlasmid.Data.Id)
		require.Equal(t, expected.Data.Type, receivedPlasmid.Data.Type)
		require.Equal(t, expected.Data.Attributes.Name, receivedPlasmid.Data.Attributes.Name)
		require.Equal(t, expected.Data.Attributes.Sequence, receivedPlasmid.Data.Attributes.Sequence)
		require.Equal(t, expected.Data.Attributes.CreatedBy, receivedPlasmid.Data.Attributes.CreatedBy)
		require.ElementsMatch(t, expected.Data.Attributes.Publications, receivedPlasmid.Data.Attributes.Publications)
	case <-timeoutCtx.Done():
		t.Fatal("Timeout waiting for published message")
	}
}
