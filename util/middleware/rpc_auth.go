package middleware

import (
	"context"
	"gitee.com/kelvins-io/common/json"
	"gitee.com/kelvins-io/common/log"
	"gitee.com/kelvins-io/kelvins/config/setting"
	"gitee.com/kelvins-io/kelvins/util/rpc_helper"
	grpcAuth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"time"
)

func GetRPCAuthDialOptions(conf *setting.RPCAuthSettingS) (opts []grpc.DialOption) {
	if conf != nil {
		if conf.Token != "" {
			opts = append(opts, grpc.WithPerRPCCredentials(RPCPerCredentials(conf.Token)))
		}
		if !conf.TransportSecurity {
			opts = append(opts, grpc.WithInsecure())
		}
	}
	return
}

type RPCPerAuthInterceptor struct {
	errLogger             log.LoggerContextIface
	tokenValidityDuration time.Duration
}

func NewRPCPerAuthInterceptor(errLogger log.LoggerContextIface) *RPCPerAuthInterceptor {
	return &RPCPerAuthInterceptor{errLogger: errLogger}
}

func (i *RPCPerAuthInterceptor) StreamServerInterceptor(conf *setting.RPCAuthSettingS) grpc.StreamServerInterceptor {
	if conf.ExpireSecond > 0 {
		i.tokenValidityDuration = time.Duration(conf.ExpireSecond) * time.Second
	}
	return grpcAuth.StreamServerInterceptor(i.checkFunc(conf))
}

func (i *RPCPerAuthInterceptor) UnaryServerInterceptor(conf *setting.RPCAuthSettingS) grpc.UnaryServerInterceptor {
	if conf.ExpireSecond > 0 {
		i.tokenValidityDuration = time.Duration(conf.ExpireSecond) * time.Second
	}
	return grpcAuth.UnaryServerInterceptor(i.checkFunc(conf))
}

func (i *RPCPerAuthInterceptor) checkFunc(conf *setting.RPCAuthSettingS) func(ctx context.Context) (context.Context, error) {
	return func(ctx context.Context) (context.Context, error) {
		if conf == nil || len(conf.Token) == 0 {
			return ctx, nil
		}

		authInfo, err := checkToken(ctx, conf.Token, time.Now(), i.tokenValidityDuration)
		if err != nil {
			if i.errLogger != nil {
				requestMeta := rpc_helper.GetRequestMetadata(ctx)
				i.errLogger.Errorf(ctx, "AuthInterceptor requestMeta: %v, checkFunc: %v err: %v", json.MarshalToStringNoError(requestMeta), authInfo, err)
			}
		}

		switch status.Code(err) {
		case codes.OK:
		case codes.Unauthenticated:
		case codes.PermissionDenied:
		default:
		}
		if conf.TransportSecurity {
			err = nil
		}

		return ctx, err
	}
}
