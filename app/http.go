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
	"gitee.com/kelvins-io/kelvins/util/kprocess"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

func RunHTTPApplication(application *kelvins.HTTPApplication) {
	if application.Name == "" {
		logging.Fatal("Application name can't not be empty")
	}
	application.Type = kelvins.AppTypeHttp

	err := runHTTP(application)
	if err != nil {
		logging.Fatalf("App.RunHTTP err: %v", err)
	}

	appPrepareForceExit()
	// Wait for connections to drain.
	err = application.HttpServer.Shutdown(context.Background())
	if err != nil {
		logging.Fatalf("App.HttpServer Shutdown err: %v", err)
	}
	err = appShutdown(application.Application)
	if err != nil {
		logging.Fatalf("App.appShutdown err: %v", err)
	}
	logging.Info("App appShutdown over")
}

func runHTTP(httpApp *kelvins.HTTPApplication) error {

	// 1. load config
	err := config.LoadDefaultConfig(httpApp.Application)
	if err != nil {
		return err
	}
	if httpApp.LoadConfig != nil {
		err = httpApp.LoadConfig()
		if err != nil {
			return err
		}
	}

	// 2. init application
	err = initApplication(httpApp.Application)
	if err != nil {
		return err
	}

	// 3. setup vars
	err = setupHTTPVars(httpApp)
	if err != nil {
		return err
	}
	if httpApp.SetupVars != nil {
		err = httpApp.SetupVars()
		if err != nil {
			return fmt.Errorf("httpApp.SetupVars err: %v", err)
		}
	}

	// 4. set init service port
	var flagPort int64
	if httpApp.Port > 0 { // use self define port to start process
		flagPort = httpApp.Port
	} else {
		flagPort = int64(util.RandInt(50000, 60000))
	}
	currentPort := strconv.Itoa(int(flagPort))

	// 5. get etcd service port
	etcdServerUrls := config.GetEtcdV3ServerURLs()
	if etcdServerUrls == "" {
		return fmt.Errorf("can't not found env '%s' ", config.ENV_ETCDV3_SERVER_URLS)
	}
	serviceLB := slb.NewService(etcdServerUrls, httpApp.Name)
	serviceConfigClient := etcdconfig.NewServiceConfigClient(serviceLB)
	serviceConfig, err := serviceConfigClient.GetConfig(currentPort)
	if err != nil && err != etcdconfig.ErrServiceConfigKeyNotExist {
		return fmt.Errorf("serviceConfig.GetConfig err: %v ,sequence(%v)", err, currentPort)
	}
	if serviceConfig != nil && serviceConfig.ServicePort == currentPort {
		return fmt.Errorf("serviceConfig.GetConfig sequence(%v) exist", currentPort)
	}
	err = serviceConfigClient.WriteConfig(currentPort, etcdconfig.Config{
		ServiceVersion: kelvins.Version,
		ServicePort:    currentPort,
	})
	if err != nil {
		return fmt.Errorf("serviceConfig.WriteConfig err: %v", err)
	}
	kelvins.ServerSetting.EndPoint = ":" + currentPort

	// 6. register http
	var handler http.Handler
	if httpApp.RegisterHttpGinEngine != nil {
		var httpGinEng *gin.Engine
		httpGinEng, err = httpApp.RegisterHttpGinEngine()
		if err != nil {
			return fmt.Errorf("httpApp.RegisterHttpGinEngine err: %v", err)
		}
		if httpGinEng != nil {
			logging.Info("http handler selected [gin]")
			handler = httpGinEng
		}
	} else {
		httpApp.Mux = setup.NewServerMux()
		if httpApp.RegisterHttpRoute != nil {
			err = httpApp.RegisterHttpRoute(httpApp.Mux)
			if err != nil {
				return fmt.Errorf("httpApp.RegisterHttpRoute err: %v", err)
			}
		}
		logging.Info("http handler selected [http.ServeMux]")
		handler = httpApp.Mux
	}
	if handler == nil {
		return fmt.Errorf("no http handler??? ")
	}

	httpApp.HttpServer = setup.NewHttpServer(
		handler,
		httpApp.TlsConfig,
		kelvins.ServerSetting,
	)

	// 7. register  event producer
	if httpApp.EventServer != nil {
		logging.Info("Start event server consume")
		// subscribe event
		if httpApp.RegisterEventProducer != nil {
			err := httpApp.RegisterEventProducer(httpApp.EventServer)
			if err != nil {
				return err
			}
		}
		// start event server
		err = httpApp.EventServer.Start()
		if err != nil {
			return err
		}
		logging.Info("Start event server")
	}

	// 8. start server
	logging.Infof("Start http server listen %s\n", kelvins.ServerSetting.EndPoint)
	network := "tcp"
	if kelvins.ServerSetting.Network != "" {
		network = kelvins.ServerSetting.Network
	}
	kp := new(kprocess.KProcess)
	ln, err := kp.Listen(network, kelvins.ServerSetting.EndPoint, kelvins.PIDFile)
	if err != nil {
		return fmt.Errorf("KProcess Listen %s err: %v", network, err)
	}
	go func() {
		err = httpApp.HttpServer.Serve(ln)
		if err != nil {
			logging.Fatalf("HttpServer serve err: %v", err)
		}
	}()

	<-kp.Exit()

	return nil
}

func setupHTTPVars(httpApp *kelvins.HTTPApplication) error {
	err := setupCommonVars(httpApp.Application)
	if err != nil {
		return err
	}

	httpApp.TraceLogger, err = log.GetAccessLogger("http.trace")
	if err != nil {
		return fmt.Errorf("kelvinslog.GetAccessLogger: %v", err)
	}

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

		httpApp.EventServer = eventServer
		return nil
	}

	return nil
}
