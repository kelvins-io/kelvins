package grpc_interceptor

import (
	"context"
	"fmt"
	"gitee.com/kelvins-io/common/json"
	"gitee.com/kelvins-io/common/log"
	"gitee.com/kelvins-io/kelvins"
	"gitee.com/kelvins-io/kelvins/internal/vars"
	"gitee.com/kelvins-io/kelvins/util/rpc_helper"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"os"
	"regexp"
	"runtime/debug"
	"time"
)

type AppServerInterceptor struct {
	accessLogger, errLogger log.LoggerContextIface
	debug                   bool
}

func NewAppServerInterceptor(debug bool, accessLogger, errLogger log.LoggerContextIface) *AppServerInterceptor {
	return &AppServerInterceptor{accessLogger: accessLogger, errLogger: errLogger, debug: debug}
}

func (i *AppServerInterceptor) Metadata(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	if methodIgnore(info.FullMethod) {
		return handler(ctx, req)
	}
	i.handleMetadata(ctx)
	return handler(ctx, req)
}

// Logger add app info in ctx.
func (i *AppServerInterceptor) Logger(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	if methodIgnore(info.FullMethod) {
		return handler(ctx, req)
	}
	incomeTime := time.Now()
	requestMeta := rpc_helper.GetRequestMetadata(ctx)
	var outcomeTime time.Time
	var resp interface{}
	var err error
	defer func() {
		outcomeTime = time.Now()
		i.echoStatistics(ctx, incomeTime, outcomeTime)
		// unary interceptor record req resp err
		if err != nil {
			if i.errLogger != nil {
				s, _ := status.FromError(err)
				i.errLogger.Errorf(
					ctx,
					"grpc access response err：%s, grpc method: %s, requestMeta: %v, outcomeTime: %v, handleTime: %f/s, req: %s, response：%s, details: %s",
					s.Err().Error(),
					info.FullMethod,
					json.MarshalToStringNoError(requestMeta),
					outcomeTime.Format(kelvins.ResponseTimeLayout),
					outcomeTime.Sub(incomeTime).Seconds(),
					json.MarshalToStringNoError(req),
					json.MarshalToStringNoError(resp),
					json.MarshalToStringNoError(s.Details()),
				)
			}
		} else {
			if i.debug && i.accessLogger != nil {
				i.accessLogger.Infof(
					ctx,
					"grpc access response ok, grpc method: %s, requestMeta: %v, outcomeTime: %v, handleTime: %f/s, req: %s, response: %s",
					info.FullMethod,
					json.MarshalToStringNoError(requestMeta),
					outcomeTime.Format(kelvins.ResponseTimeLayout),
					outcomeTime.Sub(incomeTime).Seconds(),
					json.MarshalToStringNoError(req),
					json.MarshalToStringNoError(resp),
				)
			}
		}
	}()

	resp, err = handler(ctx, req)
	return resp, err
}

func (i *AppServerInterceptor) StreamMetadata(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	if methodIgnore(info.FullMethod) {
		return handler(srv, ss)
	}
	i.handleMetadata(ss.Context())
	return handler(srv, ss)
}

// StreamLogger is experimental function
func (i *AppServerInterceptor) StreamLogger(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	if methodIgnore(info.FullMethod) {
		return handler(srv, ss)
	}
	incomeTime := time.Now()
	requestMeta := rpc_helper.GetRequestMetadata(ss.Context())
	var err error
	defer func() {
		outcomeTime := time.Now()
		i.echoStatistics(ss.Context(), incomeTime, outcomeTime)
		if err != nil {
			if i.errLogger != nil {
				s, _ := status.FromError(err)
				// stream interceptor only record error
				i.errLogger.Errorf(
					ss.Context(),
					"grpc access stream handle err：%s, grpc method: %s, requestMeta: %v, outcomeTime: %v, handleTime: %f/s, details: %s",
					s.Err().Error(),
					info.FullMethod,
					json.MarshalToStringNoError(requestMeta),
					outcomeTime.Format(kelvins.ResponseTimeLayout),
					outcomeTime.Sub(incomeTime).Seconds(),
					json.MarshalToStringNoError(s.Details()),
				)
			}
		} else {
			if i.debug && i.accessLogger != nil {
				i.accessLogger.Infof(
					ss.Context(),
					"grpc access stream handle ok, grpc method: %s, requestMeta: %v, outcomeTime: %v, handleTime: %f/s",
					info.FullMethod,
					json.MarshalToStringNoError(requestMeta),
					outcomeTime.Format(kelvins.ResponseTimeLayout),
					outcomeTime.Sub(incomeTime).Seconds(),
				)
			}
		}
	}()

	err = handler(srv, newStreamWrapper(ss.Context(), i.accessLogger, i.errLogger, ss, info, requestMeta, i.debug))
	return err
}

// Recovery recovers GRPC panic.
func (i *AppServerInterceptor) Recovery(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	requestMeta := rpc_helper.GetRequestMetadata(ctx)
	defer func() {
		if e := recover(); e != nil {
			if i.errLogger != nil {
				i.errLogger.Errorf(ctx, "grpc panic err: %v, grpc method: %s，requestMeta: %v, req: %s, stack: %s",
					e, info.FullMethod, json.MarshalToStringNoError(requestMeta), json.MarshalToStringNoError(req), string(debug.Stack()[:]))
			}
		}
	}()

	return handler(ctx, req)
}

// RecoveryStream is experimental function
func (i *AppServerInterceptor) RecoveryStream(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	requestMeta := rpc_helper.GetRequestMetadata(ss.Context())
	defer func() {
		if e := recover(); e != nil {
			if i.errLogger != nil {
				i.errLogger.Errorf(ss.Context(), "grpc stream panic err: %v, grpc method: %s, requestMeta: %v, stack: %s",
					e, info.FullMethod, json.MarshalToStringNoError(requestMeta), string(debug.Stack()[:]))
			}
		}
	}()

	return handler(srv, newStreamRecoverWrapper(ss.Context(), i.errLogger, ss, info, requestMeta, i.debug))
}

func (i *AppServerInterceptor) handleMetadata(ctx context.Context) (rId string) {
	// request id
	existRequestId, requestId := getRPCRequestId(ctx)
	if !existRequestId {
		// set request id to server
		md := metadata.Pairs(kelvins.RPCMetadataRequestId, requestId)
		ctx = metadata.NewOutgoingContext(ctx, md)
	}

	// return client info
	header := metadata.New(map[string]string{
		kelvins.RPCMetadataRequestId:   requestId,
		kelvins.RPCMetadataServiceName: kelvins.AppName,
		kelvins.RPCMetadataPowerBy:     "kelvins/rpc " + vars.Version,
	})
	if i.debug {
		header.Set(kelvins.RPCMetadataServiceNode, getRPCNodeInfo())
	}
	grpc.SetHeader(ctx, header)

	return requestId
}

func (i *AppServerInterceptor) echoStatistics(ctx context.Context, incomeTime, outcomeTime time.Time) {
	handleTime := fmt.Sprintf("%f/s", outcomeTime.Sub(incomeTime).Seconds())
	md := metadata.Pairs(kelvins.RPCMetadataResponseTime, outcomeTime.Format(kelvins.ResponseTimeLayout), kelvins.RPCMetadataHandleTime, handleTime)
	grpc.SetTrailer(ctx, md)
}

func getRPCNodeInfo() (nodeInfo string) {
	nodeInfo = fmt.Sprintf("%v:%v(%v)", vars.ServiceIp, vars.ServicePort, hostName)
	return
}

var (
	hostName, _ = os.Hostname()
)

func getRPCRequestId(ctx context.Context) (ok bool, requestId string) {
	ok = false
	md, exist := metadata.FromIncomingContext(ctx)
	if exist {
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

type streamRecoverWrapper struct {
	accessLogger, errLogger log.LoggerContextIface
	ss                      grpc.ServerStream
	ctx                     context.Context
	info                    *grpc.StreamServerInfo
	requestMeta             *rpc_helper.RequestMeta
	debug                   bool
}

func newStreamRecoverWrapper(ctx context.Context,
	errLogger log.LoggerContextIface,
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	requestMeta *rpc_helper.RequestMeta,
	debug bool) *streamRecoverWrapper {
	return &streamRecoverWrapper{ctx: ctx, errLogger: errLogger, ss: ss, info: info, requestMeta: requestMeta, debug: debug}
}
func (s *streamRecoverWrapper) SetHeader(md metadata.MD) error  { return s.ss.SetHeader(md) }
func (s *streamRecoverWrapper) SendHeader(md metadata.MD) error { return s.ss.SendHeader(md) }
func (s *streamRecoverWrapper) SetTrailer(md metadata.MD)       { s.ss.SetTrailer(md) }
func (s *streamRecoverWrapper) Context() context.Context        { return s.ss.Context() }
func (s *streamRecoverWrapper) SendMsg(m interface{}) error {
	defer func() {
		if e := recover(); e != nil {
			if s.errLogger != nil {
				s.errLogger.Errorf(s.ctx, "grpc stream/send panic err: %v, grpc method: %s, requestMeta: %v, data: %v, stack: %s",
					e, s.info.FullMethod, json.MarshalToStringNoError(s.requestMeta), json.MarshalToStringNoError(m), string(debug.Stack()[:]))
			}
		}
	}()
	return s.ss.SendMsg(m)
}
func (s *streamRecoverWrapper) RecvMsg(m interface{}) error {
	defer func() {
		if e := recover(); e != nil {
			if s.errLogger != nil {
				s.errLogger.Errorf(s.ctx, "grpc stream/recv panic err: %v, grpc method: %s, requestMeta: %v, data: %v, stack: %s",
					e, s.info.FullMethod, json.MarshalToStringNoError(s.requestMeta), json.MarshalToStringNoError(m), string(debug.Stack()[:]))
			}
		}
	}()
	return s.ss.RecvMsg(m)
}

type streamWrapper struct {
	accessLogger, errLogger log.LoggerContextIface
	ss                      grpc.ServerStream
	ctx                     context.Context
	info                    *grpc.StreamServerInfo
	requestMeta             *rpc_helper.RequestMeta
	debug                   bool
}

func newStreamWrapper(ctx context.Context,
	accessLogger, errLogger log.LoggerContextIface,
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	requestMeta *rpc_helper.RequestMeta,
	debug bool) *streamWrapper {
	return &streamWrapper{ctx: ctx, accessLogger: accessLogger, errLogger: errLogger, ss: ss, info: info, requestMeta: requestMeta, debug: debug}
}
func (s *streamWrapper) SetHeader(md metadata.MD) error  { return s.ss.SetHeader(md) }
func (s *streamWrapper) SendHeader(md metadata.MD) error { return s.ss.SendHeader(md) }
func (s *streamWrapper) SetTrailer(md metadata.MD)       { s.ss.SetTrailer(md) }
func (s *streamWrapper) Context() context.Context        { return s.ss.Context() }
func (s *streamWrapper) SendMsg(m interface{}) error {
	if methodIgnore(s.info.FullMethod) {
		return s.ss.SendMsg(m)
	}
	var err error
	incomeTime := time.Now()
	var outcomeTime time.Time
	defer func() {
		outcomeTime = time.Now()
		if err != nil {
			if s.errLogger != nil {
				sts, _ := status.FromError(err)
				s.errLogger.Errorf(
					s.ctx,
					"grpc stream/send err：%s, grpc method: %s, requestMeta: %v, outcomeTime: %v, handleTime: %f/s, data: %s, details: %s",
					sts.Err().Error(),
					s.info.FullMethod,
					json.MarshalToStringNoError(s.requestMeta),
					outcomeTime.Format(kelvins.ResponseTimeLayout),
					outcomeTime.Sub(incomeTime).Seconds(),
					json.MarshalToStringNoError(m),
					json.MarshalToStringNoError(sts.Details()),
				)
			}
		} else {
			if s.debug && s.accessLogger != nil {
				s.accessLogger.Infof(
					s.ctx,
					"grpc stream/send ok, grpc method: %s, requestMeta: %v, outcomeTime: %v, handleTime: %f/s, data: %s",
					s.info.FullMethod,
					json.MarshalToStringNoError(s.requestMeta),
					outcomeTime.Format(kelvins.ResponseTimeLayout),
					outcomeTime.Sub(incomeTime).Seconds(),
					json.MarshalToStringNoError(m),
				)
			}
		}
	}()

	err = s.ss.SendMsg(m)
	return err
}
func (s *streamWrapper) RecvMsg(m interface{}) error {
	if methodIgnore(s.info.FullMethod) {
		return s.ss.RecvMsg(m)
	}
	var err error
	incomeTime := time.Now()
	var outcomeTime time.Time
	defer func() {
		outcomeTime = time.Now()
		if err != nil {
			if s.errLogger != nil {
				sts, _ := status.FromError(err)
				s.errLogger.Errorf(
					s.ctx,
					"grpc stream/recv err：%s, grpc method: %s, requestMeta: %v, outcomeTime: %v, handleTime: %f/s, data: %s, details: %s",
					sts.Err().Error(),
					s.info.FullMethod,
					json.MarshalToStringNoError(s.requestMeta),
					outcomeTime.Format(kelvins.ResponseTimeLayout),
					outcomeTime.Sub(incomeTime).Seconds(),
					json.MarshalToStringNoError(m),
					json.MarshalToStringNoError(sts.Details()),
				)
			}
		} else {
			if s.debug && s.accessLogger != nil {
				s.accessLogger.Infof(
					s.ctx,
					"grpc stream/recv ok, grpc method: %s, requestMeta: %v, outcomeTime: %v, handleTime: %f/s, data: %s",
					s.info.FullMethod,
					json.MarshalToStringNoError(s.requestMeta),
					outcomeTime.Format(kelvins.ResponseTimeLayout),
					outcomeTime.Sub(incomeTime).Seconds(),
					json.MarshalToStringNoError(m),
				)
			}
		}
	}()

	err = s.ss.RecvMsg(m)
	return err
}

func methodIgnore(fullMethod string) (ignore bool) {
	ignore = ignoreStreamMethod.MatchString(fullMethod)
	return ignore
}

var (
	ignoreStreamMethod = regexp.MustCompilePOSIX(`^/grpc\.health\..*`)
)
