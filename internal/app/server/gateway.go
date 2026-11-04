package server

import (
	"context"
	"fmt"
	"net/http"

	EQ "github.com/IBM/fp-go/v2/eq"
	F "github.com/IBM/fp-go/v2/function"
	IOE "github.com/IBM/fp-go/v2/ioeither"
	P "github.com/IBM/fp-go/v2/predicate"
	S "github.com/IBM/fp-go/v2/string"
	predord "github.com/dictyBase/fp-go-loom/predicate/ord"
	"github.com/dictyBase/go-genproto/dictybaseapis/stock"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/urfave/cli"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type (
	gatewayInput struct {
		ctx      context.Context
		grpcS    *grpc.Server
		endpoint string
	}

	multiplexedRequest struct {
		writer  http.ResponseWriter
		request *http.Request
		grpcS   *grpc.Server
		httpMux http.Handler
	}
)

var (
	isHTTP2Proto           = F.Pipe1(httpProtoMajor2, EQ.Equals(predord.IntEq))
	isHTTP2Request         = F.Pipe1(isHTTP2Proto, P.ContraMap(requestProtoMajor))
	hasGRPCContentType     = F.Pipe1(S.HasPrefix(grpcContentTypePrefix), P.ContraMap(requestContentType))
	isGRPCRequest          = F.Pipe1(isHTTP2Request, P.And(hasGRPCContentType))
	isMultiplexedGRPC      = F.Pipe1(isGRPCRequest, P.ContraMap(multiplexedHTTPRequest))
	dispatchMultiplexedReq = F.Pipe1(isMultiplexedGRPC, P.Fold(serveHTTPMux, serveGRPCServer))
)

func requestProtoMajor(r *http.Request) int {
	return r.ProtoMajor
}

func requestContentType(r *http.Request) string {
	return r.Header.Get(contentTypeHeader)
}

func multiplexedHTTPRequest(m multiplexedRequest) *http.Request {
	return m.request
}

func serveHTTPMux(m multiplexedRequest) http.ResponseWriter {
	m.httpMux.ServeHTTP(m.writer, m.request)
	return m.writer
}

func serveGRPCServer(m multiplexedRequest) http.ResponseWriter {
	m.grpcS.ServeHTTP(m.writer, m.request)
	return m.writer
}

func exitErrorWithPrefix(prefix string) func(error) error {
	return func(err error) error {
		return cli.NewExitError(
			fmt.Sprintf("%s %s", prefix, err.Error()),
			2,
		)
	}
}

func newMultiplexedHandler(
	grpcS *grpc.Server,
	httpMux http.Handler,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = dispatchMultiplexedReq(multiplexedRequest{
			writer:  w,
			request: r,
			grpcS:   grpcS,
			httpMux: httpMux,
		})
	})
}

func newServerProtocols() *http.Protocols {
	protocols := &http.Protocols{}
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true)
	return protocols
}

func newMultiplexedServer(
	in gatewayInput,
) IOE.IOEither[error, *http.Server] {
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	// The gateway registration closes its client connection when the
	// context is done, so tie that context to the http.Server shutdown.
	gwCtx, cancel := context.WithCancel(in.ctx)
	return F.Pipe2(
		IOE.TryCatchError(func() (*runtime.ServeMux, error) {
			return mux, stock.RegisterStockServiceHandlerFromEndpoint(
				gwCtx,
				mux,
				in.endpoint,
				opts,
			)
		}),
		IOE.MapLeft[*runtime.ServeMux](func(err error) error {
			cancel()
			return F.Pipe1(err, exitErrorWithPrefix("failed to register stock gateway"))
		}),
		IOE.Map[error](func(gwMux *runtime.ServeMux) *http.Server {
			srv := &http.Server{
				Handler:           newMultiplexedHandler(in.grpcS, gwMux),
				Protocols:         newServerProtocols(),
				ReadHeaderTimeout: serverReadHeaderTimeout,
			}
			srv.RegisterOnShutdown(cancel)
			return srv
		}),
	)
}
