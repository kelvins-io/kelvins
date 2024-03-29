package app

import (
	"context"
	"fmt"
	"math"
	"time"

	"gitee.com/kelvins-io/kelvins"
	"gitee.com/kelvins-io/kelvins/config/setting"
	"gitee.com/kelvins-io/kelvins/internal/config"
	"gitee.com/kelvins-io/kelvins/internal/logging"
	setupInternal "gitee.com/kelvins-io/kelvins/internal/setup"
	"gitee.com/kelvins-io/kelvins/util/client_conn"
	"gitee.com/kelvins-io/kelvins/util/grpc_interceptor"
	"gitee.com/kelvins-io/kelvins/util/kprocess"
	"gitee.com/kelvins-io/kelvins/util/middleware"
	grpcMiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
)

// RunGRPCApplication runs grpc application.
func RunGRPCApplication(application *kelvins.GRPCApplication) {
	if application == nil || application.Application == nil {
		panic("grpcApplication is nil or application is nil")
	}
	// app instance once validate
	{
		err := appInstanceOnceValidate()
		if err != nil {
			logging.Fatal(err.Error())
		}
	}

	application.Type = kelvins.AppTypeGrpc
	kelvins.GRPCAppInstance = application

	err := runGRPC(application)
	if err != nil {
		logging.Infof("grpcApp runGRPC err: %v\n", err)
	}

	appPrepareForceExit()
	// Wait for connections to drain.
	if application.HttpServer != nil {
		err = application.HttpServer.Shutdown(context.Background())
		if err != nil {
			logging.Infof("grpcApp HttpServer.Shutdown err: %v\n", err)
		}
	}
	if application.GRPCServer != nil {
		err = stopGRPC(application)
		if err != nil {
			logging.Infof("grpcApp stopGRPC err: %v\n", err)
		}
	}
	err = appShutdown(application.Application)
	if err != nil {
		logging.Infof("grpcApp appShutdown err: %v\n", err)
	}
}

// runGRPC runs grpc application.
func runGRPC(grpcApp *kelvins.GRPCApplication) error {
	var err error

	// 1. init application
	err = initApplication(grpcApp.Application)
	if err != nil {
		return err
	}
	if !appProcessNext {
		return err
	}

	// 2 init grpc vars
	err = setupGRPCVars(grpcApp)
	if err != nil {
		return err
	}

	// 3. register service port
	portEtcd, err := appRegisterServiceToEtcd(kelvins.AppTypeText[grpcApp.Type], grpcApp.Name, grpcApp.Port)
	if err != nil {
		return err
	}
	defer func() {
		err := appUnRegisterServiceToEtcd(grpcApp.Name, grpcApp.Port)
		if err != nil {
			return
		}
	}()
	grpcApp.Port = portEtcd

	// 4. register grpc and http
	if grpcApp.RegisterGRPCServer != nil {
		err = grpcApp.RegisterGRPCServer(grpcApp.GRPCServer)
		if err != nil {
			return fmt.Errorf("registerGRPCServer err: %v", err)
		}
	}
	if grpcApp.RegisterGateway != nil {
		var opts []grpc.DialOption
		opts = append(opts, grpc.WithInsecure())
		err = grpcApp.RegisterGateway(
			context.Background(),
			grpcApp.GatewayServeMux,
			fmt.Sprintf(":%d", grpcApp.Port),
			opts,
		)
		if err != nil {
			return fmt.Errorf("registerGateway err: %v", err)
		}
	}
	if grpcApp.RegisterHttpRoute != nil {
		err = grpcApp.RegisterHttpRoute(grpcApp.Mux)
		if err != nil {
			return fmt.Errorf("registerHttpRoute err: %v", err)
		}
	}

	// 5. register event producer
	if kelvins.EventServerAliRocketMQ != nil {
		logging.Info("grpcApp Start event server")
		if grpcApp.RegisterEventProducer != nil {
			appRegisterEventProducer(grpcApp.RegisterEventProducer, grpcApp.Type)
		}
		if grpcApp.RegisterEventHandler != nil {
			appRegisterEventHandler(grpcApp.RegisterEventHandler, grpcApp.Type)
		}
	}

	// 6. start server
	network := "tcp"
	if kelvins.HttpServerSetting.Network != "" {
		network = kelvins.HttpServerSetting.Network
	}
	kp := new(kprocess.KProcess)
	ln, err := kp.Listen(network, fmt.Sprintf(":%d", grpcApp.Port), kelvins.PIDFile)
	if err != nil {
		return fmt.Errorf("kprocess listen(%s:%d) pidFile(%v) err: %v", network, grpcApp.Port, kelvins.PIDFile, err)
	}
	logging.Infof("grpcApp server listen(%s:%d) \n", network, grpcApp.Port)
	serverClose := make(chan struct{})
	go func() {
		defer func() {
			close(serverClose)
		}()
		err := grpcApp.HttpServer.Serve(ln)
		if err != nil {
			logging.Infof("grpcApp HttpServer serve err: %v", err)
		}
	}()

	select {
	case <-serverClose:
	case <-kp.Exit():
	}

	return err
}

const (
	defaultWriteBufSize = 32 * 1024
	defaultReadBufSize  = 32 * 1024
)

// setupGRPCVars ...
func setupGRPCVars(grpcApp *kelvins.GRPCApplication) error {
	var err error
	grpcApp.GKelvinsLogger = kelvins.AccessLogger
	grpcApp.GSysErrLogger = kelvins.ErrLogger
	var debug bool
	environ := grpcApp.Environment
	if environ == config.DefaultEnvironmentDev || environ == config.DefaultEnvironmentTest {
		debug = true
	}
	if kelvins.RPCRateLimitSetting == nil {
		kelvins.RPCRateLimitSetting = new(setting.RPCRateLimitSettingS)
	}
	var (
		serverUnaryInterceptors  []grpc.UnaryServerInterceptor
		serverStreamInterceptors []grpc.StreamServerInterceptor
		appInterceptor           = grpc_interceptor.NewAppServerInterceptor(debug, grpcApp.GKelvinsLogger, grpcApp.GKelvinsLogger)
		authInterceptor          = middleware.NewRPCPerAuthInterceptor(grpcApp.GKelvinsLogger)
		rateLimitParam           = kelvins.RPCRateLimitSetting
		rateLimitInterceptor     = middleware.NewRPCRateLimitInterceptor(rateLimitParam.MaxConcurrent)
	)
	serverUnaryInterceptors = append(serverUnaryInterceptors, appInterceptor.Metadata)
	serverUnaryInterceptors = append(serverUnaryInterceptors, appInterceptor.Recovery)
	if rateLimitParam.MaxConcurrent > 0 {
		serverUnaryInterceptors = append(serverUnaryInterceptors, rateLimitInterceptor.UnaryServerInterceptor())
	}
	serverUnaryInterceptors = append(serverUnaryInterceptors, appInterceptor.Logger)
	if kelvins.RPCAuthSetting == nil {
		kelvins.RPCAuthSetting = new(setting.RPCAuthSettingS)
	}
	serverUnaryInterceptors = append(serverUnaryInterceptors, authInterceptor.UnaryServerInterceptor(kelvins.RPCAuthSetting))
	if len(grpcApp.UnaryServerInterceptors) > 0 {
		serverUnaryInterceptors = append(serverUnaryInterceptors, grpcApp.UnaryServerInterceptors...)
	}
	serverStreamInterceptors = append(serverStreamInterceptors, appInterceptor.StreamMetadata)
	serverStreamInterceptors = append(serverStreamInterceptors, appInterceptor.RecoveryStream)
	if rateLimitParam.MaxConcurrent > 0 {
		serverStreamInterceptors = append(serverStreamInterceptors, rateLimitInterceptor.StreamServerInterceptor())
	}
	serverStreamInterceptors = append(serverStreamInterceptors, appInterceptor.StreamLogger)
	serverStreamInterceptors = append(serverStreamInterceptors, authInterceptor.StreamServerInterceptor(kelvins.RPCAuthSetting))
	if len(grpcApp.StreamServerInterceptors) > 0 {
		serverStreamInterceptors = append(serverStreamInterceptors, grpcApp.StreamServerInterceptors...)
	}

	var serverOptions []grpc.ServerOption
	serverOptions = append(serverOptions, grpcMiddleware.WithUnaryServerChain(serverUnaryInterceptors...))
	serverOptions = append(serverOptions, grpcMiddleware.WithStreamServerChain(serverStreamInterceptors...))
	keepaliveParams := keepalive.ServerParameters{
		MaxConnectionIdle:     5 * time.Hour,                // 空闲连接在持续一段时间后关闭
		MaxConnectionAge:      time.Duration(math.MaxInt64), // 连接的最长持续时间
		MaxConnectionAgeGrace: time.Duration(math.MaxInt64), // 最长持续时间后的 加成期，超过这个时间后强制关闭
		Time:                  2 * time.Hour,                // 服务端在这段时间后没有看到活动RPC，将给客户端发送PING
		Timeout:               20 * time.Second,             // 服务端发送PING后等待客户端应答时间，超过将关闭
	}
	keepEnforcementPolicy := keepalive.EnforcementPolicy{
		MinTime:             5 * time.Minute, // 客户端发送PING前 应该等待的最短时间
		PermitWithoutStream: true,            // 为true表示及时没有活动RPC，服务端也允许保活，为false表示客户端在没有活动RPC时发送PING将导致GoAway
	}
	serverOptions = append(serverOptions, grpc.KeepaliveParams(keepaliveParams), grpc.KeepaliveEnforcementPolicy(keepEnforcementPolicy))
	writeBufSize := grpc.WriteBufferSize(defaultWriteBufSize)
	readBufSize := grpc.ReadBufferSize(defaultReadBufSize)
	serverOptions = append(serverOptions, writeBufSize, readBufSize)
	if grpcApp.NumServerWorkers > 0 {
		serverOptions = append(serverOptions, grpc.NumStreamWorkers(grpcApp.NumServerWorkers))
	}
	// grpc app server option
	serverOptions = append(serverOptions, grpcApp.ServerOptions...)
	// server worker
	{
		cg := kelvins.RPCServerParamsSetting
		// rpc server goroutine worker num default 0
		if cg != nil && cg.NumServerWorkers > 0 {
			serverOptions = append(serverOptions, grpc.NumStreamWorkers(uint32(cg.NumServerWorkers)))
		}
		// connection time is rawConn deadline default 120s
		if cg != nil && cg.ConnectionTimeout > 0 {
			serverOptions = append(serverOptions, grpc.ConnectionTimeout(time.Duration(cg.ConnectionTimeout)*time.Second))
		}
	}
	// keep alive limit client
	{
		cg := kelvins.RPCServerKeepaliveEnforcementPolicySetting
		if cg != nil && cg.ClientMinIntervalTime > 0 {
			keepEnforcementPolicy.MinTime = time.Duration(cg.ClientMinIntervalTime) * time.Second
		}
		if cg != nil && cg.PermitWithoutStream {
			keepEnforcementPolicy.PermitWithoutStream = cg.PermitWithoutStream
		}
		if cg != nil {
			serverOptions = append(serverOptions, grpc.KeepaliveEnforcementPolicy(keepEnforcementPolicy))
		}
	}
	// keep alive
	{
		cg := kelvins.RPCServerKeepaliveParamsSetting
		if cg != nil && cg.MaxConnectionIdle > 0 {
			keepaliveParams.MaxConnectionIdle = time.Duration(cg.MaxConnectionIdle) * time.Second
		}
		if cg != nil && cg.PingClientIntervalTime > 0 {
			keepaliveParams.Time = time.Duration(cg.PingClientIntervalTime) * time.Second
		}
		if cg != nil {
			serverOptions = append(serverOptions, grpc.KeepaliveParams(keepaliveParams))
		}
	}
	// client rpc keep alive
	{
		cg := kelvins.RPCClientKeepaliveParamsSetting
		pingServerTime := 6 * time.Minute
		permitWithoutStream := true
		if cg != nil && cg.PingServerIntervalTime > 0 {
			pingServerTime = time.Duration(cg.PingServerIntervalTime) * time.Second
		}
		if cg != nil && cg.PermitWithoutStream {
			permitWithoutStream = cg.PermitWithoutStream
		}
		if cg != nil {
			opts := []grpc.DialOption{
				grpc.WithKeepaliveParams(keepalive.ClientParameters{
					Time:                pingServerTime,      // 客户端在这段时间之后如果没有活动的RPC，客户端将给服务器发送PING
					Timeout:             20 * time.Second,    // 连接服务端后等待一段时间后没有收到响应则关闭连接
					PermitWithoutStream: permitWithoutStream, // 允许客户端在没有活动RPC的情况下向服务端发送PING
				}),
			}
			client_conn.RPCClientDialOptionAppend(opts)
		}
	}
	// transport buffer
	{
		cg := kelvins.RPCTransportBufferSetting
		if cg != nil {
			const kb = 1024
			if cg.ServerReadBufSizeKB > 0 {
				serverOptions = append(serverOptions, grpc.ReadBufferSize(cg.ServerReadBufSizeKB*kb))
			}
			if cg.ServerWriteBufSizeKB > 0 {
				serverOptions = append(serverOptions, grpc.WriteBufferSize(cg.ServerWriteBufSizeKB*kb))
			}
			if cg.ClientReadBufSizeKB > 0 {
				client_conn.RPCClientDialOptionAppend([]grpc.DialOption{grpc.WithReadBufferSize(cg.ClientReadBufSizeKB * kb)})
			}
			if cg.ClientWriteBufSizeKB > 0 {
				client_conn.RPCClientDialOptionAppend([]grpc.DialOption{grpc.WithWriteBufferSize(cg.ClientWriteBufSizeKB * kb)})
			}
		}
	}

	// health server
	grpcApp.GRPCServer = setupInternal.NewGRPC(serverOptions)
	if grpcApp.GRPCServer != nil {
		if kelvins.RPCServerParamsSetting != nil && !kelvins.RPCServerParamsSetting.DisableHealthServer {
			grpcApp.HealthServer = &kelvins.GRPCHealthServer{Server: health.NewServer()}
			healthpb.RegisterHealthServer(grpcApp.GRPCServer, grpcApp.HealthServer)
			if grpcApp.RegisterGRPCHealthHandle != nil {
				go func() {
					grpcApp.RegisterGRPCHealthHandle(grpcApp.HealthServer)
				}()
			}
		}
		healthCheck := true
		if kelvins.RPCServerParamsSetting != nil && kelvins.RPCServerParamsSetting.DisableClientDialHealthCheck {
			healthCheck = false
		}
		if !healthCheck {
			client_conn.RPCClientDialOptionAppend([]grpc.DialOption{grpc.WithDisableHealthCheck()})
		}
	}
	grpcApp.GatewayServeMux = setupInternal.NewGateway()
	grpcApp.Mux = setupInternal.NewGatewayServerMux(grpcApp.GatewayServeMux, debug)
	if kelvins.HttpServerSetting == nil {
		kelvins.HttpServerSetting = new(setting.HttpServerSettingS)
	}
	kelvins.HttpServerSetting.SetAddr(fmt.Sprintf(":%d", grpcApp.Port))
	grpcApp.HttpServer = setupInternal.NewHttpServer(
		setupInternal.GRPCHandlerFunc(grpcApp.GRPCServer, grpcApp.Mux, kelvins.HttpServerSetting),
		grpcApp.TlsConfig,
		kelvins.HttpServerSetting,
	)
	// queue
	err = setupCommonQueue(nil)
	if err != nil {
		return err
	}

	return nil
}

func stopGRPC(grpcApp *kelvins.GRPCApplication) error {
	if grpcApp.HealthServer != nil {
		grpcApp.HealthServer.Shutdown()
	}
	grpcApp.GRPCServer.GracefulStop()

	return nil
}
