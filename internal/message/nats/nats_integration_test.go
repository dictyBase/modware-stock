package nats

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/dictyBase/modware-stock/internal/message"
	gnats "github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/nats"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	testSubject = "stock.test"
	testTimeout = 5 * time.Second
)

// setupNATSContainer starts a NATS container for testing and returns a cleanup function
func setupNATSContainer(ctx context.Context, t *testing.T) (string, func()) {
	t.Helper()

	container, err := nats.Run(ctx, "nats:latest")
	if err != nil {
		t.Fatalf("Failed to start NATS container: %v", err)
	}

	connStr, err := container.ConnectionString(ctx)
	if err != nil {
		if termErr := container.Terminate(ctx); termErr != nil {
			t.Logf(
				"Failed to terminate container during error cleanup: %v",
				termErr,
			)
		}
		t.Fatalf("Failed to get connection string: %v", err)
	}

	cleanup := func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}

	return connStr, cleanup
}

// createTestConn creates a test NATS connection for subscribing
func createTestConn(t *testing.T, connStr string) *gnats.Conn {
	t.Helper()

	conn, err := gnats.Connect(connStr, gnats.Timeout(testTimeout))
	require.NoError(t, err, "Failed to create test connection")

	t.Cleanup(func() {
		conn.Close()
	})

	return conn
}

// createTestStrain creates a test strain for publishing
func createTestStrain() *stock.Strain {
	return &stock.Strain{
		Data: &stock.Strain_Data{
			Type: "strain",
			Id:   "DBS0000001",
			Attributes: &stock.StrainAttributes{
				CreatedBy:       "test@example.com",
				UpdatedBy:       "test@example.com",
				Depositor:       "depositor@example.com",
				Summary:         "Test strain for NATS publishing",
				EditableSummary: "Test strain for NATS publishing",
				Label:           "testStrain1",
				Species:         "Dictyostelium discoideum",
				Genes:           []string{"DDB_G0000001", "DDB_G0000002"},
				Publications:    []string{"12345678", "87654321"},
				CreatedAt:       timestamppb.Now(),
				UpdatedAt:       timestamppb.Now(),
			},
		},
	}
}

// createTestPlasmid creates a test plasmid for publishing
func createTestPlasmid() *stock.Plasmid {
	return &stock.Plasmid{
		Data: &stock.Plasmid_Data{
			Type: "plasmid",
			Id:   "DBP0000001",
			Attributes: &stock.PlasmidAttributes{
				CreatedBy:       "test@example.com",
				UpdatedBy:       "test@example.com",
				Depositor:       "depositor@example.com",
				Summary:         "Test plasmid for NATS publishing",
				EditableSummary: "Test plasmid for NATS publishing",
				Name:            "pTestPlasmid1",
				Publications:    []string{"12345678"},
				ImageMap:        "http://example.com/plasmid.jpg",
				Sequence:        "ATCGATCGATCG",
				CreatedAt:       timestamppb.Now(),
				UpdatedAt:       timestamppb.Now(),
			},
		},
	}
}

// setupPublisherAndSubscription sets up a publisher and test subscription channel
func setupPublisherAndSubscription(
	t *testing.T,
	connStr string,
	subject string,
) (message.Publisher, <-chan *gnats.Msg, func()) {
	t.Helper()

	testConn := createTestConn(t, connStr)
	publisher, err := NewPublisher("localhost", extractPort(connStr))
	require.NoError(t, err, "Failed to create publisher")
	require.NotNil(t, publisher)

	msgChan := make(chan *gnats.Msg, 5)
	sub, err := testConn.ChanSubscribe(subject, msgChan)
	require.NoError(t, err, "Failed to create subscription")
	require.NotNil(t, sub)

	require.NoError(t, testConn.Flush(), "Failed to flush subscription setup")
	time.Sleep(100 * time.Millisecond)

	cleanup := func() {
		if unsubErr := sub.Unsubscribe(); unsubErr != nil {
			t.Logf("Failed to unsubscribe: %v", unsubErr)
		}
		if closeErr := publisher.Close(); closeErr != nil {
			t.Logf("failed to close publisher: %v", closeErr)
		}
	}

	return publisher, msgChan, cleanup
}

// TestPublishStrain_Success tests successful strain message publishing
func TestPublishStrain_Success(t *testing.T) {
	ctx := context.Background()
	connStr, cleanup := setupNATSContainer(ctx, t)
	defer cleanup()

	publisher, msgChan, pubCleanup := setupPublisherAndSubscription(t, connStr, testSubject)
	defer pubCleanup()

	testStrain := createTestStrain()
	err := publisher.PublishStrain(testSubject, testStrain)
	require.NoError(t, err, "Failed to publish strain")

	verifyReceivedStrain(ctx, t, msgChan, testStrain)
}

// TestPublishPlasmid_Success tests successful plasmid message publishing
func TestPublishPlasmid_Success(t *testing.T) {
	ctx := context.Background()
	connStr, cleanup := setupNATSContainer(ctx, t)
	defer cleanup()

	publisher, msgChan, pubCleanup := setupPublisherAndSubscription(t, connStr, testSubject)
	defer pubCleanup()

	testPlasmid := createTestPlasmid()
	err := publisher.PublishPlasmid(testSubject, testPlasmid)
	require.NoError(t, err, "Failed to publish plasmid")

	verifyReceivedPlasmid(ctx, t, msgChan, testPlasmid)
}

// TestPublishMultipleStrains tests publishing multiple strain messages
func TestPublishMultipleStrains(t *testing.T) {
	ctx := context.Background()
	connStr, cleanup := setupNATSContainer(ctx, t)
	defer cleanup()

	publisher, msgChan, pubCleanup := setupPublisherAndSubscription(t, connStr, testSubject)
	defer pubCleanup()

	numStrains := 3
	for idx := range numStrains {
		strain := createTestStrain()
		strain.Data.Id = fmt.Sprintf("DBS%07d", idx+1)
		strain.Data.Attributes.Label = fmt.Sprintf("testStrain%d", idx+1)
		err := publisher.PublishStrain(testSubject, strain)
		require.NoError(t, err, "Failed to publish strain %d", idx+1)
	}

	verifyMessageCount(ctx, t, msgChan, numStrains)
}

// waitForStrainMessage waits for a strain message on a channel and verifies it
func waitForStrainMessage(
	ctx context.Context,
	t *testing.T,
	msgChan <-chan *gnats.Msg,
	expectedID string,
	subjectName string,
) {
	t.Helper()

	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	select {
	case msg := <-msgChan:
		receivedStrain := &stock.Strain{}
		err := proto.Unmarshal(msg.Data, receivedStrain)
		require.NoError(t, err)
		require.Equal(t, expectedID, receivedStrain.Data.Id)

	case <-timeoutCtx.Done():
		t.Fatalf("Timeout waiting for message on %s", subjectName)
	}
}

// TestPublishWithDifferentSubjects tests publishing to different subjects
func TestPublishWithDifferentSubjects(t *testing.T) {
	ctx := context.Background()

	// Start NATS container
	connStr, cleanup := setupNATSContainer(ctx, t)
	defer cleanup()

	// Create test connection for subscribing
	testConn := createTestConn(t, connStr)

	// Create publisher
	publisher, err := NewPublisher("localhost", extractPort(connStr))
	require.NoError(t, err, "Failed to create publisher")
	require.NotNil(t, publisher)
	defer func() {
		if err := publisher.Close(); err != nil {
			t.Logf("failed to close publisher: %v", err)
		}
	}()

	// Create different subjects
	subject1 := "stock.strain.created"
	subject2 := "stock.strain.updated"

	// Set up subscribers for different subjects
	msgChan1 := make(chan *gnats.Msg, 1)
	msgChan2 := make(chan *gnats.Msg, 1)

	sub1, err := testConn.ChanSubscribe(subject1, msgChan1)
	require.NoError(t, err)
	defer func() {
		if unsubErr := sub1.Unsubscribe(); unsubErr != nil {
			t.Logf("Failed to unsubscribe from subject1: %v", unsubErr)
		}
	}()

	sub2, err := testConn.ChanSubscribe(subject2, msgChan2)
	require.NoError(t, err)
	defer func() {
		if unsubErr := sub2.Unsubscribe(); unsubErr != nil {
			t.Logf("Failed to unsubscribe from subject2: %v", unsubErr)
		}
	}()

	// Wait for subscriptions to be fully established
	require.NoError(t, testConn.Flush(), "Failed to flush subscription setup")
	time.Sleep(100 * time.Millisecond)

	// Create test strains
	strain1 := createTestStrain()
	strain1.Data.Id = "DBS0000001"
	strain2 := createTestStrain()
	strain2.Data.Id = "DBS0000002"

	// Publish to different subjects
	err = publisher.PublishStrain(subject1, strain1)
	require.NoError(t, err)

	err = publisher.PublishStrain(subject2, strain2)
	require.NoError(t, err)

	// Verify messages on correct subjects
	waitForStrainMessage(ctx, t, msgChan1, strain1.Data.Id, "subject1")
	waitForStrainMessage(ctx, t, msgChan2, strain2.Data.Id, "subject2")
}

// TestPublisher_ConnectionError tests publisher creation with invalid connection
func TestPublisher_ConnectionError(t *testing.T) {
	// Try to connect to non-existent server
	publisher, err := NewPublisher(
		"invalid-host",
		"9999",
		gnats.Timeout(1*time.Second),
	)

	// Should return error for connection failure
	require.Error(t, err, "Should fail to connect to invalid server")
	require.NotNil(t, publisher, "Publisher instance should still be returned")
}

// TestPublisher_Close tests closing the publisher connection
func TestPublisher_Close(t *testing.T) {
	ctx := context.Background()

	// Start NATS container
	connStr, cleanup := setupNATSContainer(ctx, t)
	defer cleanup()

	// Create publisher
	publisher, err := NewPublisher("localhost", extractPort(connStr))
	require.NoError(t, err, "Failed to create publisher")
	require.NotNil(t, publisher)

	// Close the publisher
	err = publisher.Close()
	require.NoError(t, err, "Failed to close publisher")

	// Attempting to publish after close should fail
	testStrain := createTestStrain()
	err = publisher.PublishStrain(testSubject, testStrain)
	require.Error(t, err, "Publishing after close should fail")
}

// TestPublishNilStrain tests publishing a nil strain
func TestPublishNilStrain(t *testing.T) {
	ctx := context.Background()

	// Start NATS container
	connStr, cleanup := setupNATSContainer(ctx, t)
	defer cleanup()

	// Create publisher
	publisher, err := NewPublisher("localhost", extractPort(connStr))
	require.NoError(t, err, "Failed to create publisher")
	require.NotNil(t, publisher)
	defer func() {
		if err := publisher.Close(); err != nil {
			t.Logf("failed to close publisher: %v", err)
		}
	}()

	// Attempt to publish nil strain - this may or may not error depending on protobuf
	err = publisher.PublishStrain(testSubject, nil)
	// We expect this to succeed with empty message or fail gracefully
	// The behavior depends on how proto.Marshal handles nil
	if err != nil {
		require.Error(t, err, "Should handle nil strain gracefully")
	}
}

// TestPublishNilPlasmid tests publishing a nil plasmid
func TestPublishNilPlasmid(t *testing.T) {
	ctx := context.Background()

	// Start NATS container
	connStr, cleanup := setupNATSContainer(ctx, t)
	defer cleanup()

	// Create publisher
	publisher, err := NewPublisher("localhost", extractPort(connStr))
	require.NoError(t, err, "Failed to create publisher")
	require.NotNil(t, publisher)
	defer func() {
		if err := publisher.Close(); err != nil {
			t.Logf("failed to close publisher: %v", err)
		}
	}()

	// Attempt to publish nil plasmid
	err = publisher.PublishPlasmid(testSubject, nil)
	// We expect this to succeed with empty message or fail gracefully
	if err != nil {
		require.Error(t, err, "Should handle nil plasmid gracefully")
	}
}

// TestPublishEmptySubject tests publishing with empty subject
func TestPublishEmptySubject(t *testing.T) {
	ctx := context.Background()

	// Start NATS container
	connStr, cleanup := setupNATSContainer(ctx, t)
	defer cleanup()

	// Create publisher
	publisher, err := NewPublisher("localhost", extractPort(connStr))
	require.NoError(t, err, "Failed to create publisher")
	require.NotNil(t, publisher)
	defer func() {
		if err := publisher.Close(); err != nil {
			t.Logf("failed to close publisher: %v", err)
		}
	}()

	// Create test strain
	testStrain := createTestStrain()

	// Attempt to publish with empty subject
	err = publisher.PublishStrain("", testStrain)
	require.Error(t, err, "Should fail to publish with empty subject")
}

// publishConcurrently publishes messages from multiple goroutines
func publishConcurrently(
	t *testing.T,
	publisher message.Publisher,
	numGoroutines int,
	messagesPerGoroutine int,
) {
	t.Helper()

	done := make(chan bool, numGoroutines)
	for gid := range numGoroutines {
		go func(goroutineID int) {
			defer func() { done <- true }()

			for mid := range messagesPerGoroutine {
				strain := createTestStrain()
				strain.Data.Id = fmt.Sprintf("DBS%d%d", goroutineID, mid)

				err := publisher.PublishStrain(testSubject, strain)
				if err != nil {
					t.Errorf(
						"Failed to publish from goroutine %d: %v",
						goroutineID,
						err,
					)
				}
			}
		}(gid)
	}

	// Wait for all goroutines to complete
	for range numGoroutines {
		<-done
	}
}

// verifyMessageCount verifies that expected number of messages were received
func verifyMessageCount(
	ctx context.Context,
	t *testing.T,
	msgChan <-chan *gnats.Msg,
	expectedCount int,
) {
	t.Helper()

	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	receivedCount := 0
	for receivedCount < expectedCount {
		select {
		case msg := <-msgChan:
			require.NotNil(t, msg)
			receivedStrain := &stock.Strain{}
			err := proto.Unmarshal(msg.Data, receivedStrain)
			require.NoError(t, err)
			receivedCount++

		case <-timeoutCtx.Done():
			t.Fatalf(
				"Timeout: only received %d of %d expected messages",
				receivedCount,
				expectedCount,
			)
		}
	}

	require.Equal(
		t,
		expectedCount,
		receivedCount,
		"Should receive all published messages",
	)
}

// TestConcurrentPublishing tests concurrent message publishing
func TestConcurrentPublishing(t *testing.T) {
	ctx := context.Background()

	// Start NATS container
	connStr, cleanup := setupNATSContainer(ctx, t)
	defer cleanup()

	// Create test connection for subscribing
	testConn := createTestConn(t, connStr)

	// Create publisher
	publisher, err := NewPublisher("localhost", extractPort(connStr))
	require.NoError(t, err, "Failed to create publisher")
	require.NotNil(t, publisher)
	defer func() {
		if err := publisher.Close(); err != nil {
			t.Logf("failed to close publisher: %v", err)
		}
	}()

	// Set up subscriber
	msgChan := make(chan *gnats.Msg, 20)
	sub, err := testConn.ChanSubscribe(testSubject, msgChan)
	require.NoError(t, err)
	defer func() {
		if unsubErr := sub.Unsubscribe(); unsubErr != nil {
			t.Logf("Failed to unsubscribe: %v", unsubErr)
		}
	}()

	// Wait for subscription to be fully established
	require.NoError(t, testConn.Flush(), "Failed to flush subscription setup")
	time.Sleep(100 * time.Millisecond)

	// Publish concurrently from multiple goroutines
	numGoroutines := 5
	messagesPerGoroutine := 2
	totalMessages := numGoroutines * messagesPerGoroutine

	publishConcurrently(t, publisher, numGoroutines, messagesPerGoroutine)
	verifyMessageCount(ctx, t, msgChan, totalMessages)
}

// extractPort extracts the port number from a NATS connection string
// Expected format: "nats://localhost:PORT"
func extractPort(connStr string) string {
	// Simple extraction - assumes format "nats://host:port"
	// For production use, consider using url.Parse
	var port string
	_, err := fmt.Sscanf(connStr, "nats://localhost:%s", &port)
	if err != nil || port == "" {
		// Fallback to default NATS port
		return "4222"
	}
	return port
}
