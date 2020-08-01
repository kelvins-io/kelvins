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
	newctx := metadata.NewIncomingContext(ctx, md)
	return handler(newctx, req)
}

// loggingGRPC logs GRPC request.
func (i *AppInterceptor) LoggingGRPC(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	i.App.GKelvinsLogger.Infof(ctx,
		"access request, grpc method: %s, req: %s",
		info.FullMethod,
		json.MarshalToStringNoError(req),
	)
	resp, err := handler(ctx, req)
	s, _ := status.FromError(err)
	if err != nil {
		i.App.GKelvinsLogger.Infof(
			ctx,
			"access response, grpc method: %s, response err: %v, details: %v",
			info.FullMethod,
			s.Err().Error(),
			json.MarshalToStringNoError(s.Details()),
		)
	} else if kelvins.ServerSetting.IsRecordCallResponse == true {
		i.App.GKelvinsLogger.Infof(
			ctx,
			"access response, grpc method: %s, response: %s, details: %v",
			info.FullMethod,
			json.MarshalToStringNoError(resp),
			s.Details(),
		)
	} else {
		i.App.GKelvinsLogger.Infof(
			ctx,
			"access response, grpc method: %s, details: %v",
			info.FullMethod,
			s.Details(),
		)
	}

	return resp, err
}

// RecoveryGRPC recovers GRPC panic.
func (i *AppInterceptor) RecoveryGRPC(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	defer func() {
		if e := recover(); e != nil {
			debug.PrintStack()
			i.App.GSysErrLogger.Errorf(ctx, "app panic err: %v, stack: %s", e, string(debug.Stack()[:]))
		}
	}()

	return handler(ctx, req)
}

func (i *AppInterceptor) ErrorCodeGRPC(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	resp, err := handler(ctx, req)
	if err != nil {
		i.App.GSysErrLogger.Errorf(ctx, "app return err: %v, stack: %s", err, string(debug.Stack()[:]))
	}

	return resp, err
}
