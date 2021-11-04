package app

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"gitee.com/kelvins-io/kelvins/config/setting"
	"gitee.com/kelvins-io/kelvins"
	"gitee.com/kelvins-io/kelvins/internal/config"
	"gitee.com/kelvins-io/kelvins/internal/logging"
	setupInternal "gitee.com/kelvins-io/kelvins/internal/setup"
	"gitee.com/kelvins-io/kelvins/util/gin_helper"
	"gitee.com/kelvins-io/kelvins/util/kprocess"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func RunHTTPApplication(application *kelvins.HTTPApplication) {
	if application == nil || application.Application == nil {
		panic("httpApplication is nil or application is nil")
	}
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
	err = appShutdown(application.Application)
	if err != nil {
		logging.Infof("HttpApp appShutdown err: %v\n", err)
	}
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
	portEtcd, err := appRegisterServiceToEtcd(kelvins.AppTypeText[httpApp.Type], httpApp.Name, httpApp.Port)
	if err != nil {
		return err
	}
	defer func() {
		err := appUnRegisterServiceToEtcd(httpApp.Name, httpApp.Port)
		if err != nil {
			return
		}
	}()
	httpApp.Port = portEtcd

	// 4. register http
	var handler http.Handler
	debug := false
	if kelvins.ServerSetting != nil {
		switch kelvins.ServerSetting.Environment {
		case config.DefaultEnvironmentDev:
			debug = true
		case config.DefaultEnvironmentTest:
			debug = true
		default:
		}
	}
	if httpApp.RegisterHttpGinRoute != nil {
		logging.Info("httpApp http handler selected [gin]")
		ginEngineInit()
		var httpGinEng = gin.Default()
		handler = httpGinEng
		httpGinEng.Use(gin_helper.Metadata(debug))
		httpGinEng.Use(gin_helper.Cors())
		if kelvins.HttpRateLimitSetting != nil && kelvins.HttpRateLimitSetting.MaxConcurrent > 0 {
			httpGinEng.Use(gin_helper.RateLimit(kelvins.HttpRateLimitSetting.MaxConcurrent))
		}
		if debug {
			pprof.Register(httpGinEng, "/debug")
			httpGinEng.GET("/debug/metrics", ginMetricsApi)
		}
		httpGinEng.GET("/", ginIndexApi)
		httpGinEng.GET("/ping", ginPingApi)
		httpApp.RegisterHttpGinRoute(httpGinEng)
	} else {
		httpApp.Mux = setupInternal.NewServerMux(debug)
		handler = httpApp.Mux
		httpApp.Mux.HandleFunc("/", indexApi)
		httpApp.Mux.HandleFunc("/ping", pingApi)
		if httpApp.RegisterHttpRoute != nil {
			err = httpApp.RegisterHttpRoute(httpApp.Mux)
			if err != nil {
				return fmt.Errorf("registerHttpRoute err: %v", err)
			}
		}
		logging.Info("httpApp http handler selected [http.ServeMux]")
	}
	if handler == nil {
		return fmt.Errorf("no http handler??? ")
	}

	// 5. set http server
	var httpSer *http.Server
	if kelvins.HttpServerSetting == nil {
		kelvins.HttpServerSetting = new(setting.HttpServerSettingS)
	}
	kelvins.HttpServerSetting.SetAddr(fmt.Sprintf(":%d", httpApp.Port))
	if kelvins.HttpServerSetting != nil && kelvins.HttpServerSetting.SupportH2 {
		logging.Info("httpApp enable http2/h2c")
		httpSer = setupInternal.NewHttp2Server(handler, httpApp.TlsConfig, kelvins.HttpServerSetting)
	} else {
		httpSer = setupInternal.NewHttpServer(handler, httpApp.TlsConfig, kelvins.HttpServerSetting)
	}
	httpApp.HttpServer = httpSer

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
	if kelvins.HttpServerSetting.Network != "" {
		network = kelvins.HttpServerSetting.Network
	}
	kp := new(kprocess.KProcess)
	ln, err := kp.Listen(network, fmt.Sprintf(":%d", httpApp.Port), kelvins.PIDFile)
	if err != nil {
		return fmt.Errorf("kprocess listen(%s:%d) pidFile(%v) err: %v", network, httpApp.Port, kelvins.PIDFile, err)
	}
	logging.Infof("httpApp server listen(%s:%d) \n", network, httpApp.Port)
	serverClose := make(chan struct{})
	go func() {
		defer func() {
			close(serverClose)
		}()
		err = httpApp.HttpServer.Serve(ln)
		if err != nil {
			logging.Infof("httpApp HttpServer serve err: %v", err)
		}
	}()

	select {
	case <-serverClose:
	case <-kp.Exit():
	}

	return nil
}

func setupHTTPVars(httpApp *kelvins.HTTPApplication) error {
	err := setupCommonQueue(nil)
	if err != nil {
		return err
	}

	return nil
}

func ginEngineInit() {
	var accessLogWriter io.Writer = &accessInfoLogger{}
	var errLogWriter io.Writer = &accessErrLogger{}
	if kelvins.ServerSetting != nil {
		environ := kelvins.ServerSetting.Environment
		if environ == config.DefaultEnvironmentDev || environ == config.DefaultEnvironmentTest {
			if environ == config.DefaultEnvironmentDev {
				accessLogWriter = io.MultiWriter(accessLogWriter, os.Stdout)
				errLogWriter = io.MultiWriter(errLogWriter, os.Stdout)
			}
			gin.DefaultWriter = accessLogWriter
		}
	}
	gin.DefaultErrorWriter = errLogWriter

	gin.SetMode(gin.ReleaseMode) // 默认生产
	if kelvins.ServerSetting != nil {
		switch kelvins.ServerSetting.Environment {
		case config.DefaultEnvironmentDev:
			gin.SetMode(gin.DebugMode)
		case config.DefaultEnvironmentTest:
			gin.SetMode(gin.TestMode)
		case config.DefaultEnvironmentRelease, config.DefaultEnvironmentProd:
			gin.SetMode(gin.ReleaseMode)
		default:
			gin.SetMode(gin.ReleaseMode)
		}
	}
}

type accessInfoLogger struct{}

func (a *accessInfoLogger) Write(p []byte) (n int, err error) {
	if kelvins.AccessLogger != nil {
		kelvins.AccessLogger.Infof(context.Background(), "[gin-info] %s", p)
	}
	return 0, nil
}

type accessErrLogger struct{}

func (a *accessErrLogger) Write(p []byte) (n int, err error) {
	if kelvins.AccessLogger != nil {
		kelvins.AccessLogger.Errorf(context.Background(), "[gin-err] %s", p)
	}
	return 0, nil
}

func indexApi(writer http.ResponseWriter, request *http.Request) {
	writer.WriteHeader(http.StatusOK)
	writer.Write([]byte("Welcome to " + kelvins.AppName))
}

func pingApi(writer http.ResponseWriter, request *http.Request) {
	writer.WriteHeader(http.StatusOK)
	writer.Write([]byte(time.Now().Format(kelvins.ResponseTimeLayout)))
}

func ginMetricsApi(c *gin.Context) {
	promhttp.Handler().ServeHTTP(c.Writer, c.Request)
}

func ginIndexApi(c *gin.Context) {
	gin_helper.JsonResponse(c, http.StatusOK, gin_helper.SUCCESS, "Welcome to "+kelvins.AppName)
}

func ginPingApi(c *gin.Context) {
	gin_helper.JsonResponse(c, http.StatusOK, gin_helper.SUCCESS, time.Now().Format(kelvins.ResponseTimeLayout))
}
