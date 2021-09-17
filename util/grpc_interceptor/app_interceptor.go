package grpc_interceptor

import (
	"context"
	"gitee.com/kelvins-io/common/json"
	"gitee.com/kelvins-io/kelvins"
	"gitee.com/kelvins-io/kelvins/internal/config"
	"gitee.com/kelvins-io/kelvins/internal/vars"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"runtime/debug"
)

// AppInterceptor ...
type AppInterceptor struct {
	App *kelvins.GRPCApplication
}

// AppGRPC add app info in ctx.
func (i *AppInterceptor) AppGRPC(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	md.Append("X-Request-Id", uuid.New().String())
	md.Append("X-Powered-By", "kelvins/rpc "+vars.Version)
	md.Append("kelvins-service-name", i.App.Name)
	newCtx := metadata.NewIncomingContext(ctx, md)
	return handler(newCtx, req)
}

// AppGRPCStream 是否有存在的意义
// AppGRPCStream is experimental function
func (i *AppInterceptor) AppGRPCStream(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	wrapper := &grpcStreamWrapper{
		i:  i,
		ss: ss,
	}
	return handler(srv, wrapper)
}

// LoggingGRPC loggingGRPC logs GRPC request.
func (i *AppInterceptor) LoggingGRPC(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	resp, err := handler(ctx, req)
	s, _ := status.FromError(err)
	if err != nil {
		if i.App != nil && i.App.GSysErrLogger != nil {
			i.App.GSysErrLogger.Errorf(
				ctx,
				"grpc access response err：%s, grpc method: %s, req: %s, response：%s, details: %s",
				s.Err().Error(),
				info.FullMethod,
				json.MarshalToStringNoError(req),
				json.MarshalToStringNoError(resp),
				json.MarshalToStringNoError(s.Details()),
			)
		}
	} else {
		if i.App.Environment == config.DefaultEnvironmentDev || i.App.Environment == config.DefaultEnvironmentTest {
			if i.App != nil && i.App.GSysErrLogger != nil {
				i.App.GKelvinsLogger.Infof(
					ctx,
					"grpc access response ok, grpc method: %s, req: %s, response: %s",
					info.FullMethod,
					json.MarshalToStringNoError(req),
					json.MarshalToStringNoError(resp),
				)
			}
		}
	}

	return resp, err
}

// RecoveryGRPC recovers GRPC panic.
func (i *AppInterceptor) RecoveryGRPC(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	defer func() {
		if e := recover(); e != nil {
			if i.App != nil && i.App.GSysErrLogger != nil {
				i.App.GSysErrLogger.Errorf(ctx, "grpc panic err: %v, grpc method: %s，req: %s, stack: %s",
					e, info.FullMethod, json.MarshalToStringNoError(req), string(debug.Stack()[:]))
			}
		}
	}()

	return handler(ctx, req)
}

// RecoveryGRPCStream is experimental function
func (i *AppInterceptor) RecoveryGRPCStream(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	defer func() {
		if e := recover(); e != nil {
			if i.App != nil && i.App.GSysErrLogger != nil {
				i.App.GSysErrLogger.Errorf(ss.Context(), "grpc stream panic err: %v, grpc method: %s, stack: %s",
					e, info.FullMethod, string(debug.Stack()[:]))
			}
		}
	}()

	return handler(srv, ss)
}

// grpcStreamWrapper 是否有存在的意义
type grpcStreamWrapper struct {
	i  *AppInterceptor
	ss grpc.ServerStream
}

func (s *grpcStreamWrapper) SetHeader(md metadata.MD) error {
	return s.ss.SetHeader(md)
}

func (s *grpcStreamWrapper) SendHeader(md metadata.MD) error {
	return s.ss.SendHeader(md)
}

func (s *grpcStreamWrapper) SetTrailer(md metadata.MD) {
	s.ss.SetTrailer(md)
}

func (s *grpcStreamWrapper) SendMsg(m interface{}) error {
	return s.ss.SendMsg(m)
}

func (s *grpcStreamWrapper) RecvMsg(m interface{}) error {
	return s.ss.RecvMsg(m)
}

func (s *grpcStreamWrapper) Context() context.Context {
	ctx := s.ss.Context()
	md, _ := metadata.FromIncomingContext(ctx)
	md.Append("X-Request-Id", uuid.New().String())
	md.Append("X-Powered-By", "kelvins/rpc "+vars.Version)
	if s.i != nil && s.i.App != nil {
		md.Append("kelvins-service-name", s.i.App.Name)
	}
	newCtx := metadata.NewIncomingContext(ctx, md)
	return newCtx
}
