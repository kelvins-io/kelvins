package app

import (
	"flag"
	"fmt"
	"gitee.com/kelvins-io/common/env"
	"gitee.com/kelvins-io/common/event"
	"gitee.com/kelvins-io/common/log"
	"gitee.com/kelvins-io/kelvins"
	"gitee.com/kelvins-io/kelvins/internal/config"
	"gitee.com/kelvins-io/kelvins/internal/logging"
	"gitee.com/kelvins-io/kelvins/internal/service/slb"
	"gitee.com/kelvins-io/kelvins/internal/service/slb/etcdconfig"
	"gitee.com/kelvins-io/kelvins/internal/setup"
	"gitee.com/kelvins-io/kelvins/internal/util"
	"strconv"
)

func RunHTTPApplication(application *kelvins.HTTPApplication) {
	if application.Name == "" {
		logging.Fatal("Application name can't not be empty")
	}

	flag.Parse()
	application.Port = *port
	application.LoggerRootPath = *loggerPath
	application.Type = kelvins.AppTypeHttp

	err := runHTTP(application)
	if err != nil {
		logging.Fatalf("App.RunHTTP err: %v", err)
	}
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
	var currentPort int64
	if httpApp.Port > 0 { // use self define port to start process
		currentPort = httpApp.Port
	} else {
		currentPort = int64(util.RandInt(50000, 60000))
	}

	// 5. get etcd service port
	etcdServerUrls := config.GetEtcdV3ServerURLs()
	if etcdServerUrls == "" {
		return fmt.Errorf("Can't not found env '%s'", config.ENV_ETCDV3_SERVER_URLS)
	}
	serviceLB := slb.NewService(etcdServerUrls, httpApp.Name)
	serviceConfig := etcdconfig.NewServiceConfig(serviceLB)
	if env.IsDevMode() {
		finalPort := strconv.Itoa(int(currentPort))
		serviceConfigs, err := serviceConfig.GetConfigs()
		if err != nil {
			return fmt.Errorf("serviceConfig.GetConfigs err: %v", err)
		}

		currentKey := serviceConfig.GetKeyName(httpApp.Name)
		for key, value := range serviceConfigs {
			if currentKey == key {
				finalPort = value.ServicePort
				break
			}

			if value.ServicePort == finalPort {
				return fmt.Errorf("The service port is duplicated, please try again")
			}
		}

		err = serviceConfig.WriteConfig(etcdconfig.Config{
			ServiceVersion: kelvins.Version,
			ServicePort:    finalPort,
		})
		if err != nil {
			return fmt.Errorf("serviceConfig.WriteConfig err: %v", err)
		}

		kelvins.ServerSetting.EndPoint = ":" + finalPort
	} else {
		currentConfig, err := serviceConfig.GetConfig()
		if err != nil {
			return fmt.Errorf("serviceConfig.GetConfig err: %v", err)
		}

		kelvins.ServerSetting.EndPoint = ":" + currentConfig.ServicePort
	}

	// 6. register grpc and http
	httpApp.Mux = setup.NewServerMux()
	if httpApp.RegisterHttpRoute != nil {
		err = httpApp.RegisterHttpRoute(httpApp.Mux)
		if err != nil {
			return fmt.Errorf("httpApp.RegisterHttpRoute err: %v", err)
		}
	}
	httpApp.HttpServer = setup.NewHttpServer(
		httpApp.Mux,
		httpApp.TlsConfig,
		kelvins.ServerSetting,
	)

	// 7. register  event producer
	if httpApp.EventServer != nil {
		logging.Infof("Start event server consume")
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
	logging.Infof("Start http server listen %s", kelvins.ServerSetting.EndPoint)
	err = httpApp.HttpServer.ListenAndServe()
	if err != nil {
		return fmt.Errorf("HttpServer serve err: %v", err)
	}

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
