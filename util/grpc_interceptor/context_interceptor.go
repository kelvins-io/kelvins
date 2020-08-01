package grpc_interceptor

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

func UnaryCtxHandleGRPC() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, resp interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		ctx, cancel := ctxHandler(ctx)
		if cancel != nil {
			defer cancel()
		}

		return invoker(ctx, method, req, resp, cc, opts...)
	}
}

func StreamCtxHandleGRPC() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		ctx, cancel := ctxHandler(ctx)
		if cancel != nil {
			defer cancel()
		}

		return streamer(ctx, desc, cc, method, opts...)
	}
}

func ctxHandler(ctx context.Context) (context.Context, context.CancelFunc) {
	var cancel context.CancelFunc
	if _, ok := ctx.Deadline(); !ok {
		var defaultTimeout = 60 * time.Second

		ctx, cancel = context.WithTimeout(ctx, defaultTimeout)
	}

	return ctx, cancel
}
