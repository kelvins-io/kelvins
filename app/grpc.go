package app

import (
	"context"
	"flag"
	"fmt"
	"gitee.com/kelvins-io/common/env"
	"gitee.com/kelvins-io/common/log"
	"gitee.com/kelvins-io/kelvins"
	"gitee.com/kelvins-io/kelvins/internal/config"
	"gitee.com/kelvins-io/kelvins/internal/logging"
	"gitee.com/kelvins-io/kelvins/internal/service/slb"
	"gitee.com/kelvins-io/kelvins/internal/service/slb/etcdconfig"
	"gitee.com/kelvins-io/kelvins/internal/setup"
	"gitee.com/kelvins-io/kelvins/internal/util"
	"gitee.com/kelvins-io/kelvins/util/grpc_interceptor"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"google.golang.org/grpc"
	"net"
	"net/http"
	"strconv"
)

// RunGRPCApplication runs grpc application.
func RunGRPCApplication(application *kelvins.GRPCApplication) {
	if application.Name == "" {
		logging.Fatal("Application name can't not be empty")
	}

	flag.Parse()
	application.Port = *port
	application.LoggerRootPath = *loggerPath
	application.Type = kelvins.AppTypeGrpc

	err := runGRPC(application)
	if err != nil {
		logging.Fatalf("App.RunGRPC err: %v", err)
	}
}

// runGRPC runs grpc application.
func runGRPC(grpcApp *kelvins.GRPCApplication) error {
	// 1. init application
	err := initApplication(grpcApp.Application)
	if err != nil {
		return err
	}

	// 2. load config
	err = config.LoadDefaultConfig(grpcApp.Application)
	if err != nil {
		return err
	}
	if grpcApp.LoadConfig != nil {
		err = grpcApp.LoadConfig()
		if err != nil {
			return err
		}
	}

	// 3. setup vars
	err = setupGRPCVars(grpcApp)
	if err != nil {
		return err
	}
	if grpcApp.SetupVars != nil {
		err = grpcApp.SetupVars()
		if err != nil {
			return fmt.Errorf("grpcApp.SetupVars err: %v", err)
		}
	}

	// 4. set init service port
	var flagPort int64
	if grpcApp.Port > 0 { // use self define port to start process
		flagPort = grpcApp.Port
	} else if env.IsDevMode() {
		flagPort = int64(util.RandInt(50000, 60000))
	}
	currentPort := strconv.Itoa(int(flagPort))

	// 5. get etcd service port
	etcdServerUrls := config.GetEtcdV3ServerURLs()
	if etcdServerUrls == "" {
		return fmt.Errorf("Can't not found env '%s'", config.ENV_ETCDV3_SERVER_URLS)
	}
	serviceLB := slb.NewService(etcdServerUrls, grpcApp.Name)
	serviceConfig := etcdconfig.NewServiceConfig(serviceLB)
	if env.IsDevMode() {
		serviceConfigs, err := serviceConfig.GetConfigs()
		if err != nil {
			return fmt.Errorf("serviceConfig.GetConfigs err: %v", err)
		}

		currentKey := serviceConfig.GetKeyName(grpcApp.Name)
		for key, value := range serviceConfigs {
			if currentKey == key {
				currentPort = value.ServicePort
				break
			}

			if value.ServicePort == currentPort {
				return fmt.Errorf("The service port is duplicated, please try again")
			}
		}
		err = serviceConfig.WriteConfig(etcdconfig.Config{
			ServiceVersion: kelvins.Version,
			ServicePort:    currentPort,
			HttpPort:       currentPort,
		})
		if err != nil {
			return fmt.Errorf("serviceConfig.WriteConfig err: %v", err)
		}

	} else if flagPort <= 0 {
		currentConfig, err := serviceConfig.GetConfig()
		if err != nil {
			return fmt.Errorf("serviceConfig.GetConfig err: %v", err)
		}

		currentPort = currentConfig.ServicePort
	}
	kelvins.ServerSetting.EndPoint = ":" + currentPort

	// 6. register grpc and http
	if grpcApp.RegisterGRPCServer != nil {
		err = grpcApp.RegisterGRPCServer(grpcApp.GRPCServer)
		if err != nil {
			return fmt.Errorf("grpcApp.RegisterGRPCServer err: %v", err)
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
			return fmt.Errorf("grpcApp.RegisterGateway err: %v", err)
		}
	}
	if grpcApp.RegisterHttpRoute != nil {
		err = grpcApp.RegisterHttpRoute(grpcApp.Mux)
		if err != nil {
			return fmt.Errorf("grpcApp.RegisterHttpRoute err: %v", err)
		}
	}

	// 7. apollo hot update listen
	//config.TriggerApolloHotUpdateListen(grpcApp.Application)

	// run mux
	go func() {
		http.ListenAndServe(":"+currentPort, grpcApp.Mux)
	}()

	// 8. start server
	logging.Infof("Start http server listen %s", kelvins.ServerSetting.EndPoint)
	conn, err := net.Listen("tcp", kelvins.ServerSetting.EndPoint)
	if err != nil {
		return fmt.Errorf("TCP Listen err: %v", err)
	}
	err = grpcApp.GRPCServer.Serve(conn)
	if err != nil {
		return fmt.Errorf("HttpServer serve err: %v", err)
	}

	return nil
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
	serverInterceptors = append(serverInterceptors, appInterceptor.ErrorCodeGRPC)
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
		setup.GRPCHandlerFunc(grpcApp.GRPCServer, grpcApp.Mux),
		grpcApp.TlsConfig,
		kelvins.ServerSetting,
	)

	return nil
}
