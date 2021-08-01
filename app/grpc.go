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
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"google.golang.org/grpc"
	"strconv"
)

// RunGRPCApplication runs grpc application.
func RunGRPCApplication(application *kelvins.GRPCApplication) {
	if application.Name == "" {
		logging.Fatal("Application name can't not be empty")
	}
	application.Type = kelvins.AppTypeGrpc

	err := runGRPC(application)
	if err != nil {
		logging.Infof("gRPC App.RunGRPC err: %v", err)
	}

	appPrepareForceExit()
	// Wait for connections to drain.
	err = application.HttpServer.Shutdown(context.Background())
	if err != nil {
		logging.Infof("gRPC App HttpServer.Shutdown err: %v", err)
	}
	application.GRPCServer.Stop()
	err = appShutdown(application.Application)
	if err != nil {
		logging.Infof("gRPC App.appShutdown err: %v", err)
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
		return fmt.Errorf("gRPC KProcess Listen %s err: %v", network, err)
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

// setupGRPCVars ...
func setupGRPCVars(grpcApp *kelvins.GRPCApplication) error {
	err := setupCommonVars(grpcApp.Application)
	if err != nil {
		return err
	}

	grpcApp.GKelvinsLogger, err = log.GetAccessLogger("grpc.access")
	if err != nil {
		return fmt.Errorf("kelvinslog.GetAccessLogger: %v", err)
	}

	grpcApp.GSysErrLogger, err = log.GetErrLogger("grpc.sys.err")
	if err != nil {
		return fmt.Errorf("kelvinslog.GetErrLogger: %v", err)
	}

	var (
		serverInterceptors []grpc.UnaryServerInterceptor
		appInterceptor     = grpc_interceptor.AppInterceptor{App: grpcApp}
	)
	serverInterceptors = append(serverInterceptors, appInterceptor.RecoveryGRPC)
	serverInterceptors = append(serverInterceptors, appInterceptor.LoggingGRPC)
	serverInterceptors = append(serverInterceptors, appInterceptor.AppGRPC)
	//serverInterceptors = append(serverInterceptors, appInterceptor.ErrorCodeGRPC)
	if len(grpcApp.UnaryServerInterceptors) > 0 {
		serverInterceptors = append(serverInterceptors, grpcApp.UnaryServerInterceptors...)
	}

	serverOptions := append(grpcApp.ServerOptions, grpc_middleware.WithUnaryServerChain(serverInterceptors...))
	grpcApp.GRPCServer, err = setup.NewGRPC(kelvins.ServerSetting, serverOptions)
	if err != nil {
		return fmt.Errorf("Setup.SetupGRPC err: %v", err)
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
