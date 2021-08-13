package middleware

import (
	"context"
	"gitee.com/kelvins-io/kelvins"
	"gitee.com/kelvins-io/kelvins/config/setting"
	grpcAuth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"time"
)

func GetRPCAuthDialOptions(conf *setting.RPCAuthSettingS) (opts []grpc.DialOption) {
	if conf != nil {
		if conf.Token != "" {
			opts = append(opts, grpc.WithPerRPCCredentials(RPCCredentials(conf.Token)))
		}
		if !conf.TransportSecurity {
			opts = append(opts, grpc.WithInsecure())
		}
	}
	return
}

type AuthInterceptor struct {
	App *kelvins.GRPCApplication
}

func (i *AuthInterceptor) StreamServerInterceptor(conf *setting.RPCAuthSettingS) grpc.StreamServerInterceptor {
	return grpcAuth.StreamServerInterceptor(i.checkFunc(conf))
}

func (i *AuthInterceptor) UnaryServerInterceptor(conf *setting.RPCAuthSettingS) grpc.UnaryServerInterceptor {
	return grpcAuth.UnaryServerInterceptor(i.checkFunc(conf))
}

func (i *AuthInterceptor) checkFunc(conf *setting.RPCAuthSettingS) func(ctx context.Context) (context.Context, error) {
	return func(ctx context.Context) (context.Context, error) {
		if conf == nil || len(conf.Token) == 0 {
			return ctx, nil
		}

		authInfo, err := checkToken(ctx, conf.Token, time.Now())
		if err != nil {
			i.App.GSysErrLogger.Errorf(ctx, "AuthInterceptor checkFunc %v err %v", authInfo, err)
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
