package client_conn

import (
	"context"
	"fmt"
	"gitee.com/kelvins-io/kelvins/internal/config"
	"gitee.com/kelvins-io/kelvins/internal/service/slb"
	"gitee.com/kelvins-io/kelvins/internal/service/slb/etcdconfig"
	"gitee.com/kelvins-io/kelvins/util/grpc_interceptor"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"strings"
)

var (
	opts []grpc.DialOption
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
func (c *ConnClient) GetConn(ctx context.Context) (*grpc.ClientConn, error) {
	target := fmt.Sprintf("%s:///%s", kelvinsScheme, c.ServerName)
	return grpc.DialContext(
		ctx,
		target,
		opts...,
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

func init() {
	opts = append(opts, grpc.WithInsecure())
	opts = append(opts, grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`))
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
}
