package app

import (
	"context"
	"fmt"
	"gitee.com/kelvins-io/common/event"
	"gitee.com/kelvins-io/common/log"
	"gitee.com/kelvins-io/kelvins"
	"gitee.com/kelvins-io/kelvins/internal/config"
	"gitee.com/kelvins-io/kelvins/internal/logging"
	"gitee.com/kelvins-io/kelvins/internal/service/slb"
	"gitee.com/kelvins-io/kelvins/internal/service/slb/etcdconfig"
	"gitee.com/kelvins-io/kelvins/internal/setup"
	"gitee.com/kelvins-io/kelvins/internal/util"
	"gitee.com/kelvins-io/kelvins/util/grpc_interceptor"
	"gitee.com/kelvins-io/kelvins/util/kprocess"
	"gitee.com/kelvins-io/kelvins/util/middleware"
	grpcMiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"math"
	"strconv"
	"time"
)

// RunGRPCApplication runs grpc application.
func RunGRPCApplication(application *kelvins.GRPCApplication) {
	if application.Name == "" {
		logging.Fatal("Application name can't not be empty")
	}
	application.Type = kelvins.AppTypeGrpc

	err := runGRPC(application)
	if err != nil {
		logging.Infof("gRPC App.RunGRPC err: %v\n", err)
	}

	appPrepareForceExit()
	// Wait for connections to drain.
	if application.HttpServer != nil {
		err = application.HttpServer.Shutdown(context.Background())
		if err != nil {
			logging.Infof("gRPC App HttpServer.Shutdown err: %v\n", err)
		}
	}
	if application.GRPCServer != nil {
		err = stopGRPC(application)
		if err != nil {
			logging.Infof("gRPC stopGRPC err: %v\n", err)
		}
		application.GRPCServer.Stop()
	}
	err = appShutdown(application.Application)
	if err != nil {
		logging.Infof("gRPC App.appShutdown err: %v\n", err)
	}
	logging.Info("gRPC App.appShutdown over")
}

// runGRPC runs grpc application.
func runGRPC(grpcApp *kelvins.GRPCApplication) error {

	// 1. load config
	err := config.LoadDefaultConfig(grpcApp.Application)
	if err != nil {
		return err
	}
	if grpcApp.LoadConfig != nil {
		err = grpcApp.LoadConfig()
		if err != nil {
			return err
		}
	}

	// 2. init application
	err = initApplication(grpcApp.Application)
	if err != nil {
		return err
	}

	// 3. setup vars
	err = setupCommonVars(grpcApp.Application)
	if err != nil {
		return err
	}
	err = setupGRPCVars(grpcApp)
	if err != nil {
		return err
	}
	if grpcApp.SetupVars != nil {
		err = grpcApp.SetupVars()
		if err != nil {
			return fmt.Errorf("App.SetupVars err: %v", err)
		}
	}

	// startup control
	next, err := startUpControl(kelvins.PIDFile)
	if err != nil {
		return err
	}
	if !next {
		return nil
	}

	// 4. set init service port
	var flagPort int64
	if grpcApp.Port > 0 { // use self define port to start process
		flagPort = grpcApp.Port
	} else {
		flagPort = int64(util.RandInt(50000, 60000))
	}
	currentPort := strconv.Itoa(int(flagPort))

	// 5. get etcd service port
	etcdServerUrls := config.GetEtcdV3ServerURLs()
	if etcdServerUrls == "" {
		return fmt.Errorf("can't not found env '%s'", config.ENV_ETCDV3_SERVER_URLS)
	}
	serviceLB := slb.NewService(etcdServerUrls, grpcApp.Name)
	serviceConfigClient := etcdconfig.NewServiceConfigClient(serviceLB)
	serviceConfig, err := serviceConfigClient.GetConfig(currentPort)
	if err != nil && err != etcdconfig.ErrServiceConfigKeyNotExist {
		return fmt.Errorf("serviceConfig.GetConfig err: %v sequence(%v)", err, currentPort)
	}
	if serviceConfig != nil && serviceConfig.ServicePort == currentPort {
		return fmt.Errorf("serviceConfig.GetConfig currentPort(%v) exist", currentPort)
	}
	err = serviceConfigClient.WriteConfig(currentPort, etcdconfig.Config{
		ServiceVersion: kelvins.Version,
		ServicePort:    currentPort,
	})
	if err != nil {
		return fmt.Errorf("serviceConfig.WriteConfig err: %v", err)
	}
	kelvins.ServerSetting.EndPoint = ":" + currentPort

	// 6. register grpc and http
	if grpcApp.RegisterGRPCServer != nil {
		err = grpcApp.RegisterGRPCServer(grpcApp.GRPCServer)
		if err != nil {
			return fmt.Errorf("App.RegisterGRPCServer err: %v", err)
		}
	}
	if grpcApp.RegisterGateway != nil {
		var opts []grpc.DialOption
		opts = append(opts, grpc.WithInsecure())
		err = grpcApp.RegisterGateway(
			context.Background(),
			grpcApp.GatewayServeMux,
			kelvins.ServerSetting.EndPoint,
			opts,
		)
		if err != nil {
			return fmt.Errorf("App.RegisterGateway err: %v", err)
		}
	}
	if grpcApp.RegisterHttpRoute != nil {
		err = grpcApp.RegisterHttpRoute(grpcApp.Mux)
		if err != nil {
			return fmt.Errorf("App.RegisterHttpRoute err: %v", err)
		}
	}

	// 7. register event producer
	if grpcApp.EventServer != nil {
		logging.Info("gRPC Start event server consume")
		// subscribe event
		if grpcApp.RegisterEventProducer != nil {
			err := grpcApp.RegisterEventProducer(grpcApp.EventServer)
			if err != nil {
				return err
			}
		}
		// start event server
		err = grpcApp.EventServer.Start()
		if err != nil {
			return err
		}
		logging.Info("gRPC Start event server")
	}

	// 8. start server
	logging.Infof("gRPC Start http server listen %s\n", kelvins.ServerSetting.EndPoint)
	network := "tcp"
	if kelvins.ServerSetting.Network != "" {
		network = kelvins.ServerSetting.Network
	}
	kp := new(kprocess.KProcess)
	ln, err := kp.Listen(network, kelvins.ServerSetting.EndPoint, kelvins.PIDFile)
	if err != nil {
		return fmt.Errorf("gRPC KProcess Listen %s%s err: %v", network, kelvins.ServerSetting.EndPoint, err)
	}
	go func() {
		err = grpcApp.HttpServer.Serve(ln)
		if err != nil {
			logging.Infof("gRPC HttpServer serve err: %v", err)
		}
	}()

	<-kp.Exit()

	return err
}

const (
	defaultWriteBufSize = 32 * 1024
	defaultReadBufSize  = 32 * 1024
)

// setupGRPCVars ...
func setupGRPCVars(grpcApp *kelvins.GRPCApplication) error {
	var err error
	grpcApp.GKelvinsLogger, err = log.GetAccessLogger("grpc.access")
	if err != nil {
		return fmt.Errorf("kelvinslog.GetAccessLogger: %v", err)
	}

	grpcApp.GSysErrLogger, err = log.GetErrLogger("grpc.sys.err")
	if err != nil {
		return fmt.Errorf("kelvinslog.GetErrLogger: %v", err)
	}

	var (
		serverUnaryInterceptors  []grpc.UnaryServerInterceptor
		serverStreamInterceptors []grpc.StreamServerInterceptor
		appInterceptor           = grpc_interceptor.AppInterceptor{App: grpcApp}
		authInterceptor          = middleware.AuthInterceptor{App: grpcApp}
	)
	serverUnaryInterceptors = append(serverUnaryInterceptors, appInterceptor.RecoveryGRPC)
	serverUnaryInterceptors = append(serverUnaryInterceptors, authInterceptor.UnaryServerInterceptor(kelvins.RPCAuthSetting))
	serverUnaryInterceptors = append(serverUnaryInterceptors, appInterceptor.LoggingGRPC)
	serverUnaryInterceptors = append(serverUnaryInterceptors, appInterceptor.AppGRPC)
	serverUnaryInterceptors = append(serverUnaryInterceptors, appInterceptor.ErrorCodeGRPC)
	if len(grpcApp.UnaryServerInterceptors) > 0 {
		serverUnaryInterceptors = append(serverUnaryInterceptors, grpcApp.UnaryServerInterceptors...)
	}
	serverStreamInterceptors = append(serverStreamInterceptors, authInterceptor.StreamServerInterceptor(kelvins.RPCAuthSetting))
	if len(grpcApp.StreamServerInterceptors) > 0 {
		serverStreamInterceptors = append(serverStreamInterceptors, grpcApp.StreamServerInterceptors...)
	}
	// keep alive limit client
	keepEnforcementPolicyOpt := grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
		MinTime:             20 * time.Second,
		PermitWithoutStream: true,
	})
	// keep alive
	keepaliveParamsOpt := grpc.KeepaliveParams(keepalive.ServerParameters{
		MaxConnectionIdle:     time.Duration(math.MaxInt64),
		MaxConnectionAge:      time.Duration(math.MaxInt64),
		MaxConnectionAgeGrace: time.Duration(math.MaxInt64),
		Time:                  2 * time.Hour,
		Timeout:               20 * time.Second,
	})
	writeBufSize := grpc.WriteBufferSize(defaultWriteBufSize)
	readBufSize := grpc.ReadBufferSize(defaultReadBufSize)
	var serverOptions []grpc.ServerOption
	serverOptions = append(serverOptions, grpcMiddleware.WithUnaryServerChain(serverUnaryInterceptors...))
	serverOptions = append(serverOptions, grpcMiddleware.WithStreamServerChain(serverStreamInterceptors...))
	serverOptions = append(serverOptions, keepaliveParamsOpt, keepEnforcementPolicyOpt)
	serverOptions = append(serverOptions, writeBufSize, readBufSize)
	if grpcApp.NumServerWorkers > 0 {
		serverOptions = append(serverOptions, grpc.NumStreamWorkers(grpcApp.NumServerWorkers))
	}
	serverOptions = append(serverOptions, grpcApp.ServerOptions...)
	grpcApp.GRPCServer, err = setup.NewGRPC(kelvins.ServerSetting, serverOptions)
	if err != nil {
		return fmt.Errorf("Setup.SetupGRPC err: %v", err)
	}
	if grpcApp.GRPCServer != nil && !grpcApp.DisableHealthCheck {
		grpcApp.HealthServer = health.NewServer()
		healthpb.RegisterHealthServer(grpcApp.GRPCServer, grpcApp.HealthServer)
		if grpcApp.RegisterHealthServer != nil {
			go func() {
				grpcApp.RegisterHealthServer(grpcApp.HealthServer)
			}()
		}
	}
	grpcApp.GatewayServeMux = setup.NewGateway()
	grpcApp.Mux = setup.NewGatewayServerMux(grpcApp.GatewayServeMux)
	grpcApp.HttpServer = setup.NewHttpServer(
		setup.GRPCHandlerFunc(grpcApp.GRPCServer, grpcApp.Mux, kelvins.ServerSetting),
		grpcApp.TlsConfig,
		kelvins.ServerSetting,
	)

	// init event server
	if kelvins.AliRocketMQSetting != nil && kelvins.AliRocketMQSetting.InstanceId != "" {
		logger, err := log.GetBusinessLogger("event")
		if err != nil {
			return err
		}

		// new event server
		eventServer, err := event.NewEventServer(&event.Config{
			BusinessName: kelvins.AliRocketMQSetting.BusinessName,
			RegionId:     kelvins.AliRocketMQSetting.RegionId,
			AccessKey:    kelvins.AliRocketMQSetting.AccessKey,
			SecretKey:    kelvins.AliRocketMQSetting.SecretKey,
			InstanceId:   kelvins.AliRocketMQSetting.InstanceId,
			HttpEndpoint: kelvins.AliRocketMQSetting.HttpEndpoint,
		}, logger)
		if err != nil {
			return err
		}

		grpcApp.EventServer = eventServer
		return nil
	}

	return nil
}

func stopGRPC(grpcApp *kelvins.GRPCApplication) error {
	if grpcApp.HealthServer != nil {
		grpcApp.HealthServer.Shutdown()
	}
	return nil
}
