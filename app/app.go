package app

import (
	"flag"
	"fmt"
	"gitee.com/kelvins-io/common/log"
	"gitee.com/kelvins-io/kelvins"
	"gitee.com/kelvins-io/kelvins/internal/config"
	"gitee.com/kelvins-io/kelvins/internal/logging"
	"gitee.com/kelvins-io/kelvins/internal/service/slb"
	"gitee.com/kelvins-io/kelvins/internal/service/slb/etcdconfig"
	"gitee.com/kelvins-io/kelvins/setup"
	"gitee.com/kelvins-io/kelvins/util/goroutine"
	"gitee.com/kelvins-io/kelvins/util/startup"
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
	kelvins.ServerName = application.Name

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

	err := log.InitGlobalConfig(loggerPath, loggerLevel, application.Name)
	if err != nil {
		return fmt.Errorf("log.InitGlobalConfig: %v", err)
	}

	return nil
}

// setupCommonVars setup application global vars.
func setupCommonVars(application *kelvins.Application) error {
	var err error
	if kelvins.ServerSetting != nil {
		if kelvins.ServerSetting.PIDFile != "" {
			kelvins.PIDFile = kelvins.ServerSetting.PIDFile
		} else {
			wd, _ := os.Getwd()
			kelvins.PIDFile = fmt.Sprintf("%s/%s.pid", wd, application.Name)
		}
	}

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

	kelvins.ErrLogger, err = log.GetErrLogger("err")
	if err != nil {
		return err
	}

	kelvins.BusinessLogger, err = log.GetBusinessLogger("business")
	if err != nil {
		return err
	}

	kelvins.AccessLogger, err = log.GetAccessLogger("access")
	if err != nil {
		return err
	}

	return nil
}

// appCloseChOne is AppCloseCh sync.Once
var appCloseChOne sync.Once

func appShutdown(application *kelvins.Application) error {
	if !execStopFunc {
		return nil
	}
	appCloseChOne.Do(func() {
		close(kelvins.AppCloseCh)
	})

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

	if application.StopFunc != nil {
		err := application.StopFunc()
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
	if !execStopFunc {
		return
	}
	time.AfterFunc(30*time.Second, func() {
		logging.Info("App Graceful shutdown timed out, force exit")
		os.Exit(1)
	})
}

var execStopFunc bool

func startUpControl(pidFile string) (next bool, err error) {
	next, err = startup.ParseCliCommand(pidFile)
	if next {
		execStopFunc = true
	}
	return
}
