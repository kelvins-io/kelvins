package app

import (
	"context"
	"flag"
	"fmt"
	"gitee.com/kelvins-io/common/log"
	"gitee.com/kelvins-io/kelvins"
	"gitee.com/kelvins-io/kelvins/internal/config"
	"gitee.com/kelvins-io/kelvins/internal/logging"
	"gitee.com/kelvins-io/kelvins/internal/service/slb"
	"gitee.com/kelvins-io/kelvins/internal/service/slb/etcdconfig"
	"gitee.com/kelvins-io/kelvins/internal/vars"
	"gitee.com/kelvins-io/kelvins/setup"
	"gitee.com/kelvins-io/kelvins/util/goroutine"
	"gitee.com/kelvins-io/kelvins/util/startup"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	DefaultLoggerRootPath = "./logs"
	DefaultLoggerLevel    = "info"
)

var (
	flagLoggerLevel = flag.String("logger_level", "", "set logger level eg: debug,warn,error,info")
	flagLoggerPath  = flag.String("logger_path", "", "set logger root path eg: /tmp/kelvins-app")
	flagEnv         = flag.String("env", "", "set exec environment eg: dev,test,prod")
)

func initApplication(application *kelvins.Application) error {
	flag.Parse()
	if application.Name == "" {
		logging.Fatal("Application name can't not be empty")
	}
	kelvins.ServerName = application.Name

	// 1 show app version
	showAppVersion(application)

	// 2. load app config
	err := config.LoadDefaultConfig(application)
	if err != nil {
		return err
	}

	// 3 setup app proc
	setupApplicationProcess(application)

	// 4 startup control
	next, err := startUpControl(kelvins.PIDFile)
	if err != nil {
		return err
	}
	if !next {
		close(appCloseCh)
		<-kelvins.AppCloseCh
		return nil
	}

	// 5 init user config
	if application.LoadConfig != nil {
		err = application.LoadConfig()
		if err != nil {
			return err
		}
	}

	// 6 init system vars
	// init logger vars
	loggerPath := DefaultLoggerRootPath
	if kelvins.LoggerSetting != nil && kelvins.LoggerSetting.RootPath != "" {
		loggerPath = kelvins.LoggerSetting.RootPath
	}
	if application.LoggerRootPath != "" {
		loggerPath = application.LoggerRootPath
	}
	if *flagLoggerPath != "" {
		loggerPath = *flagLoggerPath
	}
	application.LoggerRootPath = loggerPath

	loggerLevel := DefaultLoggerLevel
	if kelvins.LoggerSetting != nil && kelvins.LoggerSetting.Level != "" {
		loggerLevel = kelvins.LoggerSetting.Level
	}
	if application.LoggerLevel != "" {
		loggerLevel = application.LoggerLevel
	}
	if *flagLoggerLevel != "" {
		loggerLevel = *flagLoggerLevel
	}
	application.LoggerLevel = loggerLevel

	// init environment
	environment := config.DefaultEnvironmentProd
	if kelvins.ServerSetting != nil && kelvins.ServerSetting.Environment != "" {
		environment = kelvins.ServerSetting.Environment
	}
	if application.Environment != "" {
		environment = application.Environment
	}
	if *flagEnv != "" {
		environment = *flagEnv
	}
	application.Environment = environment

	// 7 init log
	err = log.InitGlobalConfig(loggerPath, loggerLevel, application.Name)
	if err != nil {
		return fmt.Errorf("log.InitGlobalConfig: %v", err)
	}

	// 8. setup vars
	// setup app vars
	err = setupCommonVars(application)
	if err != nil {
		return err
	}
	// setup user vars
	if application.SetupVars != nil {
		err = application.SetupVars()
		if err != nil {
			return fmt.Errorf("application.SetupVars err: %v", err)
		}
	}
	return nil
}

// setup AppCloseCh pid
func setupApplicationProcess(application *kelvins.Application) {
	kelvins.AppCloseCh = appCloseCh
	vars.Version = kelvins.Version
	if kelvins.ServerSetting != nil {
		if kelvins.ServerSetting.PIDFile != "" {
			kelvins.PIDFile = kelvins.ServerSetting.PIDFile
		} else {
			wd, _ := os.Getwd()
			kelvins.PIDFile = fmt.Sprintf("%s/%s.pid", wd, application.Name)
		}
	}
}

// setupCommonVars setup application global vars.
func setupCommonVars(application *kelvins.Application) error {
	var err error
	if kelvins.MysqlSetting != nil && kelvins.MysqlSetting.Host != "" {
		kelvins.MysqlSetting.LoggerLevel = application.LoggerLevel
		kelvins.MysqlSetting.Environment = application.Environment
		kelvins.GORM_DBEngine, err = setup.NewMySQLWithGORM(kelvins.MysqlSetting)
		kelvins.XORM_DBEngine, err = setup.NewMySQLWithXORM(kelvins.MysqlSetting)
		if err != nil {
			return err
		}
	}

	if kelvins.MongoDBSetting != nil && kelvins.MongoDBSetting.Uri != "" {
		kelvins.MongoDBClient, err = setup.NewMongoDBClient(kelvins.MongoDBSetting)
		if err != nil {
			return err
		}
	}

	if kelvins.RedisSetting != nil && kelvins.RedisSetting.Host != "" {
		kelvins.RedisConn, err = setup.NewRedis(kelvins.RedisSetting)
		if err != nil {
			return err
		}
	}

	if kelvins.GPoolSetting != nil && kelvins.GPoolSetting.JobChanLen > 0 && kelvins.GPoolSetting.WorkerNum > 0 {
		kelvins.GPool = goroutine.NewPool(kelvins.GPoolSetting.WorkerNum, kelvins.GPoolSetting.JobChanLen)
	}

	kelvins.FrameworkLogger, err = log.GetCustomLogger("framework", "framework")
	if err != nil {
		return err
	}
	vars.FrameworkLogger = kelvins.FrameworkLogger

	kelvins.ErrLogger, err = log.GetErrLogger("err")
	if err != nil {
		return err
	}
	vars.ErrLogger = kelvins.ErrLogger

	kelvins.BusinessLogger, err = log.GetBusinessLogger("business")
	if err != nil {
		return err
	}
	vars.BusinessLogger = kelvins.BusinessLogger

	kelvins.AccessLogger, err = log.GetAccessLogger("access")
	if err != nil {
		return err
	}
	vars.AccessLogger = kelvins.AccessLogger

	return nil
}

// appCloseChOne is appCloseCh sync.Once
var appCloseChOne sync.Once
var appCloseCh = make(chan struct{})

func appShutdown(application *kelvins.Application) error {
	if !appProcessNext {
		return nil
	}
	appCloseChOne.Do(func() {
		close(appCloseCh)
	})
	if application.StopFunc != nil {
		err := application.StopFunc()
		if err != nil {
			return err
		}
	}
	if application.Type == kelvins.AppTypeHttp || application.Type == kelvins.AppTypeGrpc {
		etcdServerUrls := config.GetEtcdV3ServerURLs()
		if etcdServerUrls == "" {
			return fmt.Errorf("can't not found env '%s'\n", config.ENV_ETCDV3_SERVER_URLS)
		}
		serviceLB := slb.NewService(etcdServerUrls, application.Name)
		serviceConfigClient := etcdconfig.NewServiceConfigClient(serviceLB)
		sequence := strings.TrimPrefix(kelvins.ServerSetting.EndPoint, ":")
		err := serviceConfigClient.ClearConfig(sequence)
		if err != nil {
			return fmt.Errorf("serviceConfigClient ClearConfig err: %v, key: %v\n", err, serviceConfigClient.GetKeyName(application.Name, sequence))
		}
	}
	if kelvins.GPool != nil {
		kelvins.GPool.Release()
		kelvins.GPool.WaitAll()
	}
	if kelvins.RedisConn != nil {
		err := kelvins.RedisConn.Close()
		if err != nil {
			return err
		}
	}
	if kelvins.GORM_DBEngine != nil {
		err := kelvins.GORM_DBEngine.Close()
		if err != nil {
			return err
		}
	}
	if kelvins.MongoDBClient != nil {
		err := kelvins.MongoDBClient.Close(context.Background())
		if err != nil {
			return err
		}
	}

	return nil
}

func appPrepareForceExit() {
	// Make sure to set a deadline on exiting the process
	// after upg.Exit() is closed. No new upgrades can be
	// performed if the parent doesn't exit.
	if !appProcessNext {
		return
	}
	time.AfterFunc(30*time.Second, func() {
		logging.Info("App Graceful shutdown timed out, force exit")
		os.Exit(1)
	})
}

var appProcessNext bool

func startUpControl(pidFile string) (next bool, err error) {
	next, err = startup.ParseCliCommand(pidFile)
	if next {
		appProcessNext = true
	}
	return
}

func showAppVersion(app *kelvins.Application) {
	var logo = `%20__%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20___%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%0A%2F%5C%20%5C%20%20%20%20%20%20%20%20%20%20%20%20%20%2F%5C_%20%5C%20%20%20%20%20%20%20%20%20%20%20%20%20__%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%0A%5C%20%5C%20%5C%2F'%5C%20%20%20%20%20%20%20__%5C%2F%2F%5C%20%5C%20%20%20%20__%20%20__%20%2F%5C_%5C%20%20%20%20%20___%20%20%20%20%20%20____%20%20%0A%20%5C%20%5C%20%2C%20%3C%20%20%20%20%20%2F'__%60%5C%5C%20%5C%20%5C%20%20%2F%5C%20%5C%2F%5C%20%5C%5C%2F%5C%20%5C%20%20%2F'%20_%20%60%5C%20%20%20%2F'%2C__%5C%20%0A%20%20%5C%20%5C%20%5C%5C%60%5C%20%20%2F%5C%20%20__%2F%20%5C_%5C%20%5C_%5C%20%5C%20%5C_%2F%20%7C%5C%20%5C%20%5C%20%2F%5C%20%5C%2F%5C%20%5C%20%2F%5C__%2C%20%60%5C%0A%20%20%20%5C%20%5C_%5C%20%5C_%5C%5C%20%5C____%5C%2F%5C____%5C%5C%20%5C___%2F%20%20%5C%20%5C_%5C%5C%20%5C_%5C%20%5C_%5C%5C%2F%5C____%2F%0A%20%20%20%20%5C%2F_%2F%5C%2F_%2F%20%5C%2F____%2F%5C%2F____%2F%20%5C%2F__%2F%20%20%20%20%5C%2F_%2F%20%5C%2F_%2F%5C%2F_%2F%20%5C%2F___%2F%20`
	var version = `[Major Version：%v Type：%v]`
	var remote = `┌───────────────────────────────────────────────────┐
│ [Gitee] https://gitee.com/kelvins-io/kelvins      │
│ [GitHub] https://github.com/kelvins-io/kelvins    │
└───────────────────────────────────────────────────┘`
	fmt.Println("based on")
	logoS, _ := url.QueryUnescape(logo)
	fmt.Println(logoS)
	fmt.Println("")
	fmt.Println(fmt.Sprintf(version, kelvins.Version, kelvins.AppTypeText[app.Type]))

	fmt.Println("")
	fmt.Println(remote)
	fmt.Println("Go Go Go ==>", app.Name)
}
