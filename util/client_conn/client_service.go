package client_conn

import (
	"context"
	"errors"
	"fmt"
	"gitee.com/kelvins-io/kelvins/internal/config"
	"gitee.com/kelvins-io/kelvins/internal/service/slb"
	"gitee.com/kelvins-io/kelvins/internal/service/slb/etcdconfig"
	"gitee.com/kelvins-io/kelvins/util/grpc_interceptor"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"strings"
)

const LOCALHOST = "127.0.0.1"

type Conn struct {
	ServerName string
	ServerPort string
}

func NewConn(serviceName string) (*Conn, error) {
	serviceNames := strings.Split(serviceName, "-")
	if len(serviceNames) < 1 {
		return nil, errors.New("NewConn.serviceNames is empty")
	}
	etcdServerUrls := config.GetEtcdV3ServerURLs()
	if len(etcdServerUrls) == 0 {
		return nil, fmt.Errorf("Can't not found env '%s'", config.ENV_ETCDV3_SERVER_URLS)
	}
	serviceLB := slb.NewService(etcdServerUrls, serviceName)
	serviceConfig := etcdconfig.NewServiceConfig(serviceLB)
	currentConfig, err := serviceConfig.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("serviceConfig.GetConfig err: %v", err)
	}

	return &Conn{
		ServerName: serviceName,
		ServerPort: currentConfig.ServicePort,
	}, nil
}

func (c *Conn) GetConn(ctx context.Context) (*grpc.ClientConn, error) {
	var opts []grpc.DialOption

	target := c.ServerName + ":" + c.ServerPort

	opts = append(opts, grpc.WithInsecure())
	opts = append(opts, grpc.WithUnaryInterceptor(
		grpc_middleware.ChainUnaryClient(
			grpc_interceptor.UnaryCtxHandleGRPC(),
			grpc_retry.UnaryClientInterceptor(
				grpc_retry.WithMax(2),
				grpc_retry.WithCodes(
					codes.Internal,
					codes.DeadlineExceeded,
				),
			),
		),
	))
	opts = append(opts, grpc.WithStreamInterceptor(
		grpc_middleware.ChainStreamClient(
			grpc_interceptor.StreamCtxHandleGRPC(),
		),
	))

	return grpc.DialContext(
		ctx,
		target,
		opts...,
	)
}
