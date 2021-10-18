package client_conn

import (
	"context"
	"fmt"
	"gitee.com/kelvins-io/kelvins/internal/config"
	"gitee.com/kelvins-io/kelvins/internal/logging"
	"gitee.com/kelvins-io/kelvins/internal/service/slb"
	"gitee.com/kelvins-io/kelvins/internal/service/slb/etcdconfig"
	"gitee.com/kelvins-io/kelvins/internal/vars"
	"gitee.com/kelvins-io/kelvins/util/grpc_interceptor"
	grpcMiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpcRetry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/keepalive"
	"strings"
	"sync"
	"time"
)

var (
	optsDefault []grpc.DialOption
	optsStartup []grpc.DialOption
)

var (
	mutexCreateConn sync.Map
)

type ConnClient struct {
	ServerName string
}

func NewConnClient(serviceName string) (*ConnClient, error) {
	serviceNames := strings.Split(serviceName, "-")
	if len(serviceNames) < 1 {
		return nil, fmt.Errorf("serviceNames(%v) format not contain `-` ", serviceName)
	}

	return &ConnClient{
		ServerName: serviceName,
	}, nil
}

// GetConn return a valid connection as much as possible
func (c *ConnClient) GetConn(ctx context.Context, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	conn, err := getRPCConn(c.ServerName)
	if err == nil && justConnEffective(conn) {
		return conn, nil
	}

	// prevent the same service from concurrently creating connections
	{
		v, _ := mutexCreateConn.LoadOrStore(c.ServerName, &sync.Mutex{})
		mutex := v.(*sync.Mutex)
		mutex.Lock()
		defer mutex.Unlock()
	}

	conn, err = getRPCConn(c.ServerName)
	if err == nil && justConnEffective(conn) {
		return conn, nil
	}

	// priority order: optsStartup > opts > optsDefault
	optsUse := append(optsDefault, opts...)
	target := fmt.Sprintf("%s:///%s", kelvinsScheme, c.ServerName)
	conn, err = grpc.DialContext(
		ctx,
		target,
		append(optsUse, optsStartup...)...,
	)

	if err == nil && justConnEffective(conn) {
		_ee := storageRPCConn(c.ServerName, conn)
		if _ee != nil {
			if vars.FrameworkLogger != nil {
				vars.FrameworkLogger.Errorf(ctx, "storageRPCConn(%s) err %v", c.ServerName, _ee)
			} else {
				logging.Errf("storageRPCConn(%s) err %v\n", c.ServerName, _ee)
			}
		}
	}

	return conn, err
}

// GetEndpoints the returned endpoint list may have invalid nodes
func (c *ConnClient) GetEndpoints(ctx context.Context) (endpoints []string, err error) {
	etcdServerUrls := config.GetEtcdV3ServerURLs()
	serviceLB := slb.NewService(etcdServerUrls, c.ServerName)
	serviceConfigClient := etcdconfig.NewServiceConfigClient(serviceLB)
	serviceConfigs, err := serviceConfigClient.GetConfigs()
	if err != nil {
		if vars.FrameworkLogger != nil {
			vars.FrameworkLogger.Errorf(ctx, "etcd GetConfig(%v) err %v", c.ServerName, err)
		} else {
			logging.Errf("etcd GetConfig(%v) err %v\n", c.ServerName, err)
		}
		return
	}
	for _, value := range serviceConfigs {
		var addr = fmt.Sprintf("%v:%v", c.ServerName, value.ServicePort)
		endpoints = append(endpoints, addr)
	}
	return
}

func justConnEffective(conn *grpc.ClientConn) bool {
	if conn == nil {
		return false
	}
	state := conn.GetState()
	if state == connectivity.Idle || state == connectivity.Ready {
		return true
	} else {
		conn.Close()
		return false
	}
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
	defaultWriteBufSize = 32 * 1024
	defaultReadBufSize  = 32 * 1024
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
		Time:                6 * time.Minute,  // 客户端在这段时间之后如果没有活动的RPC，客户端将给服务器发送PING
		Timeout:             20 * time.Second, // 连接服务端后等待一段时间后没有收到响应则关闭连接
		PermitWithoutStream: true,             // 允许客户端在没有活动RPC的情况下向服务端发送PING
	}))
	optsDefault = append(optsDefault, grpc.WithReadBufferSize(defaultReadBufSize))
	optsDefault = append(optsDefault, grpc.WithWriteBufferSize(defaultWriteBufSize))
}

// RPCClientDialOptionAppend only executed at boot load, so no locks are used
func RPCClientDialOptionAppend(opts []grpc.DialOption) {
	optsStartup = append(optsStartup, opts...)
}
