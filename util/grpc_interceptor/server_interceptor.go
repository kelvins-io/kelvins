package grpc_interceptor

import (
	"context"
	"fmt"
	"gitee.com/kelvins-io/common/json"
	"gitee.com/kelvins-io/kelvins"
	"gitee.com/kelvins-io/kelvins/internal/config"
	"gitee.com/kelvins-io/kelvins/internal/vars"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"runtime/debug"
	"time"
)

// AppInterceptor ...
type AppInterceptor struct {
	App *kelvins.GRPCApplication
}

// AppGRPC add app info in ctx.
func (i *AppInterceptor) AppGRPC(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	incomeTime := time.Now()
	i.handleMetadata(ctx)
	defer func() {
		outcomeTime := time.Now()
		i.statistics(ctx, incomeTime, outcomeTime)
	}()

	return handler(ctx, req)
}

// AppGRPCStream is experimental function
func (i *AppInterceptor) AppGRPCStream(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	incomeTime := time.Now()
	i.handleMetadata(ss.Context())
	defer func() {
		outcomeTime := time.Now()
		i.statistics(ss.Context(), incomeTime, outcomeTime)
	}()

	return handler(srv, ss)
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

func (i *AppInterceptor) handleMetadata(ctx context.Context) {
	// request id
	okRequestId, requestId := getRPCRequestId(ctx)
	if !okRequestId {
		// set request id to server
		md := metadata.Pairs(kelvins.RPCMetadataRequestId, requestId)
		ctx = metadata.NewOutgoingContext(ctx, md)
	}
	// return client info
	header := metadata.New(map[string]string{
		kelvins.RPCMetadataRequestId:   requestId,
		kelvins.RPCMetadataServiceName: i.App.Name,
		kelvins.RPCMetadataPowerBy:     "kelvins/rpc " + vars.Version,
	})
	grpc.SetHeader(ctx, header)
}

func (i *AppInterceptor) statistics(ctx context.Context, incomeTime, outcomeTime time.Time) {
	handleTime := fmt.Sprintf("%f/s", outcomeTime.Sub(incomeTime).Seconds())
	md := metadata.Pairs(kelvins.RPCMetadataResponseTime, outcomeTime.Format(kelvins.ResponseTimeLayout), kelvins.RPCMetadataHandleTime, handleTime)
	grpc.SetTrailer(ctx, md)
}

func getRPCRequestId(ctx context.Context) (ok bool, requestId string) {
	ok = false
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		if t, ok := md[kelvins.RPCMetadataRequestId]; ok {
			for _, e := range t {
				if e != "" {
					requestId = e
					ok = true
					break
				}
			}
		}
	}
	if requestId == "" {
		requestId = uuid.New().String()
	}
	return
}
