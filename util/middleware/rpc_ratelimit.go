package middleware

import (
	"context"
	"gitee.com/kelvins-io/common/json"
	"gitee.com/kelvins-io/kelvins/util/rpc_helper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type RPCRateLimitInterceptor struct {
	limiter Limiter
}

func NewRPCRateLimitInterceptor(maxConcurrent int) *RPCRateLimitInterceptor {
	return &RPCRateLimitInterceptor{
		limiter: NewKelvinsRateLimit(maxConcurrent),
	}
}

func (r *RPCRateLimitInterceptor) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if r.limiter.Limit() {
			requestMeta := rpc_helper.GetRequestMetadata(stream.Context())
			return status.Errorf(codes.ResourceExhausted, "%s requestMeta:%v is rejected by grpc_ratelimit middleware, please retry later.", info.FullMethod, json.MarshalToStringNoError(requestMeta))
		}
		defer func() {
			r.limiter.ReturnTicket()
		}()
		return handler(srv, stream)
	}
}

func (r *RPCRateLimitInterceptor) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if r.limiter.Limit() {
			requestMeta := rpc_helper.GetRequestMetadata(ctx)
			return nil, status.Errorf(codes.ResourceExhausted, "%s requestMeta:%v is rejected by grpc_ratelimit middleware, please retry later.", info.FullMethod, json.MarshalToStringNoError(requestMeta))
		}
		defer func() {
			r.limiter.ReturnTicket()
		}()
		return handler(ctx, req)
	}
}
