package client_conn

import (
	"context"
	"fmt"
	"gitee.com/kelvins-io/kelvins/internal/config"
	"gitee.com/kelvins-io/kelvins/internal/service/slb"
	"gitee.com/kelvins-io/kelvins/internal/service/slb/etcdconfig"
	"gitee.com/kelvins-io/kelvins/util/grpc_interceptor"
	grpcMiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpcRetry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"math"
	"strings"
	"time"
)

var (
	optsDefault []grpc.DialOption
)

type ConnClient struct {
	ServerName string
}

func NewConnClient(serviceName string) (*ConnClient, error) {
	serviceNames := strings.Split(serviceName, "-")
	if len(serviceNames) < 1 {
		return nil, fmt.Errorf("serviceNames(%v) format not contain '-'", serviceName)
	}

	return &ConnClient{
		ServerName: serviceName,
	}, nil
}

// return a valid connection as much as possible
func (c *ConnClient) GetConn(ctx context.Context, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	target := fmt.Sprintf("%s:///%s", kelvinsScheme, c.ServerName)

	return grpc.DialContext(
		ctx,
		target,
		append(optsDefault, opts...)...,
	)
}

// the returned endpoint list may have invalid nodes
func (c *ConnClient) GetEndpoints(ctx context.Context) (endpoints []string, err error) {
	etcdServerUrls := config.GetEtcdV3ServerURLs()
	serviceLB := slb.NewService(etcdServerUrls, c.ServerName)
	serviceConfigClient := etcdconfig.NewServiceConfigClient(serviceLB)
	serviceConfigs, err := serviceConfigClient.GetConfigs()
	if err != nil {
		return
	}
	for _, value := range serviceConfigs {
		endpoints = append(endpoints, value.ServicePort)
	}
	return
}

const (
	grpcServiceConfig = `{
	"loadBalancingPolicy": "round_robin",
	"healthCheckConfig": {
		"serviceName": ""
	}
}`
)

const (
	defaultWriteBufSize = 64 * 1024
	defaultReadBufSize  = 64 * 1024
)

func init() {
	optsDefault = append(optsDefault, grpc.WithInsecure())
	optsDefault = append(optsDefault, grpc.WithDefaultServiceConfig(grpcServiceConfig))
	optsDefault = append(optsDefault, grpc.WithUnaryInterceptor(
		grpcMiddleware.ChainUnaryClient(
			grpc_interceptor.UnaryCtxHandleGRPC(),
			grpcRetry.UnaryClientInterceptor(
				grpcRetry.WithMax(2),
				grpcRetry.WithCodes(
					codes.Internal,
					codes.DeadlineExceeded,
				),
			),
		),
	))
	optsDefault = append(optsDefault, grpc.WithStreamInterceptor(
		grpcMiddleware.ChainStreamClient(
			grpc_interceptor.StreamCtxHandleGRPC(),
		),
	))
	optsDefault = append(optsDefault, grpc.WithKeepaliveParams(keepalive.ClientParameters{
		Time:                time.Duration(math.MaxInt64),
		Timeout:             20 * time.Second,
		PermitWithoutStream: true,
	}))
	optsDefault = append(optsDefault, grpc.WithReadBufferSize(defaultReadBufSize))
	optsDefault = append(optsDefault, grpc.WithWriteBufferSize(defaultWriteBufSize))
}
