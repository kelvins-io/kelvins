package grpc_interceptor

import (
	"context"
	"gitee.com/kelvins-io/common/json"
	"gitee.com/kelvins-io/kelvins"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"runtime/debug"
	"strconv"
)

// AppInterceptor ...
type AppInterceptor struct {
	App *kelvins.GRPCApplication
}

// AppGRPC add app info in ctx.
func (i *AppInterceptor) AppGRPC(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	md.Append("kelvins-service-name", i.App.Name)
	md.Append("kelvins-service-type", strconv.Itoa(int(i.App.Type)))
	md.Append("kelvins-service-version", kelvins.Version)
	newCtx := metadata.NewIncomingContext(ctx, md)
	return handler(newCtx, req)
}

// loggingGRPC logs GRPC request.
func (i *AppInterceptor) LoggingGRPC(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	resp, err := handler(ctx, req)
	s, _ := status.FromError(err)
	if err != nil {
		i.App.GSysErrLogger.Errorf(
			ctx,
			"access response err：%s, grpc method: %s, req: %s, response：%s, details: %s",
			s.Err().Error(),
			info.FullMethod,
			json.MarshalToStringNoError(req),
			json.MarshalToStringNoError(resp),
			json.MarshalToStringNoError(s.Details()),
		)
	} else if kelvins.ServerSetting.IsRecordCallResponse == true {
		i.App.GKelvinsLogger.Infof(
			ctx,
			"access response ok, grpc method: %s, req: %s, response: %s",
			info.FullMethod,
			json.MarshalToStringNoError(req),
			json.MarshalToStringNoError(resp),
		)
	}

	return resp, err
}

// RecoveryGRPC recovers GRPC panic.
func (i *AppInterceptor) RecoveryGRPC(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	defer func() {
		if e := recover(); e != nil {
			debug.PrintStack()
			i.App.GSysErrLogger.Errorf(ctx, "app panic err: %v, grpc method: %s，req: %s, stack: %s", e, info.FullMethod, json.MarshalToStringNoError(req), string(debug.Stack()[:]))
		}
	}()

	return handler(ctx, req)
}
