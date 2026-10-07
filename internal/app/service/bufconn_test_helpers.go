package service

import (
	"context"
	"net"
	"testing"

	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

// newBufconnClient serves svc over a buffer connection and returns a
// client for it. The serving goroutine does not log and does not exit
// the process: Serve returns grpc.ErrServerStopped during cleanup, and
// the goroutine can outlive the test, so logging would race with the
// testing package and os.Exit would kill the test binary. The target
// carries its own passthrough scheme, so no global resolver change is
// needed — a call to resolver.SetDefaultScheme races with every other
// test in the package. Every buffer-connection test in this package
// must dial through this helper.
func newBufconnClient(t *testing.T, svc *StockService) stock.StockServiceClient {
	t.Helper()
	server := grpc.NewServer()
	stock.RegisterStockServiceServer(server, svc)
	lis := bufconn.Listen(1024 * 1024)
	go func() {
		_ = server.Serve(lis)
	}()
	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
	)
	t.Cleanup(func() {
		_ = conn.Close()
		_ = lis.Close()
		server.Stop()
	})
	if err != nil {
		t.Fatalf("expect no error creating a grpc client, received %s", err)
	}
	return stock.NewStockServiceClient(conn)
}
