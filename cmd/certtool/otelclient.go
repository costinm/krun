package main

import (
	"context"
	"net/http"

	"github.com/costinm/krun/pkg/mesh"
	"go.opentelemetry.io/contrib/instrumentation/host"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"google.golang.org/grpc"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

// TODO: use otelhttptrace to get httptrace (low level client traces)

func initOTel(ctx context.Context, kr *mesh.KRun) {
	kr.TransportWrapper = func(transport http.RoundTripper) http.RoundTripper {
		return otelhttp.NewTransport(transport)
	}
	// Host telemetry -
	host.Start()
}

func OTELGRPCClient() []grpc.DialOption {
	return []grpc.DialOption{
		grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()),
		grpc.WithStreamInterceptor(otelgrpc.StreamClientInterceptor())}
}
