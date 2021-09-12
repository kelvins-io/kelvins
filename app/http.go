package app

import (
	"context"
	"fmt"
	"gitee.com/kelvins-io/kelvins"
	"gitee.com/kelvins-io/kelvins/internal/logging"
	setupInternal "gitee.com/kelvins-io/kelvins/internal/setup"
	"gitee.com/kelvins-io/kelvins/util/kprocess"
	"github.com/gin-gonic/gin"
	"net/http"
)

func RunHTTPApplication(application *kelvins.HTTPApplication) {
	// app instance once validate
	{
		err := appInstanceOnceValidate()
		if err != nil {
			logging.Fatal(err.Error())
		}
	}

	application.Type = kelvins.AppTypeHttp
	kelvins.HttpAppInstance = application

	err := runHTTP(application)
	if err != nil {
		logging.Infof("HttpApp runHTTP err: %v\n", err)
	}

	appPrepareForceExit()
	// Wait for connections to drain.
	if application.HttpServer != nil {
		err = application.HttpServer.Shutdown(context.Background())
		if err != nil {
			logging.Infof("HttpApp HttpServer Shutdown err: %v\n", err)
		}
	}
	err = appShutdown(application.Application, application.Port)
	if err != nil {
		logging.Infof("HttpApp appShutdown err: %v\n", err)
	}
	logging.Info("HttpApp appShutdown over")
}

func runHTTP(httpApp *kelvins.HTTPApplication) error {
	var err error

	// 1. init application
	err = initApplication(httpApp.Application)
	if err != nil {
		return err
	}
	if !appProcessNext {
		return err
	}

	// 2 init http vars
	err = setupHTTPVars(httpApp)
	if err != nil {
		return err
	}

	// 3. set init service port
	portEtcd, err := appRegisterServiceToEtcd(httpApp.Name, httpApp.Port)
	if err != nil {
		return err
	}
	httpApp.Port = portEtcd

	// 4. register http
	var handler http.Handler
	if httpApp.RegisterHttpGinEngine != nil {
		var httpGinEng *gin.Engine
		httpGinEng, err = httpApp.RegisterHttpGinEngine()
		if err != nil {
			return fmt.Errorf("registerHttpGinEngine err: %v", err)
		}
		if httpGinEng != nil {
			logging.Info("httpApp http handler selected [gin]")
			handler = httpGinEng
		}
	} else {
		httpApp.Mux = setupInternal.NewServerMux()
		if httpApp.RegisterHttpRoute != nil {
			err = httpApp.RegisterHttpRoute(httpApp.Mux)
			if err != nil {
				return fmt.Errorf("registerHttpRoute err: %v", err)
			}
		}
		logging.Info("httpApp http handler selected [http.ServeMux]")
		handler = httpApp.Mux
	}
	if handler == nil {
		return fmt.Errorf("no http handler??? ")
	}

	// 5. set http server
	kelvins.ServerSetting.SetAddr(fmt.Sprintf(":%d", httpApp.Port))
	httpApp.HttpServer = setupInternal.NewHttpServer(
		handler,
		httpApp.TlsConfig,
		kelvins.ServerSetting,
	)

	// 6. register event producer
	if kelvins.EventServerAliRocketMQ != nil {
		logging.Info("httpApp start event server")
		if httpApp.RegisterEventProducer != nil {
			appRegisterEventProducer(httpApp.RegisterEventProducer, httpApp.Type)
		}
		if httpApp.RegisterEventHandler != nil {
			appRegisterEventHandler(httpApp.RegisterEventHandler, httpApp.Type)
		}
	}

	// 7. start server
	network := "tcp"
	if kelvins.ServerSetting.Network != "" {
		network = kelvins.ServerSetting.Network
	}
	kp := new(kprocess.KProcess)
	ln, err := kp.Listen(network, fmt.Sprintf(":%d", httpApp.Port), kelvins.PIDFile)
	if err != nil {
		return fmt.Errorf("kprocess listen(%s:%d) pidFile(%v) err: %v", network, httpApp.Port, kelvins.PIDFile, err)
	}
	logging.Infof("httpApp server listen(%s:%d) \n", network, httpApp.Port)
	go func() {
		err = httpApp.HttpServer.Serve(ln)
		if err != nil {
			logging.Infof("httpApp HttpServer serve err: %v", err)
		}
	}()

	<-kp.Exit()

	return nil
}

func setupHTTPVars(httpApp *kelvins.HTTPApplication) error {
	err := setupCommonQueue(nil)
	if err != nil {
		return err
	}

	return nil
}
