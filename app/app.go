package app

import (
	"flag"
	"fmt"
	"gitee.com/kelvins-io/common/log"
	"gitee.com/kelvins-io/kelvins"
	"gitee.com/kelvins-io/kelvins/internal/logging"
	"gitee.com/kelvins-io/kelvins/setup"
	"gitee.com/kelvins-io/kelvins/util/goroutine"
	"os"
	"time"
)

const (
	DefaultLoggerRootPath = "./logs"
	DefaultLoggerLevel    = "debug"
)

var (
	flagLoggerLevel = flag.String("logger_level", "", "Set Logger Level.")
	flagLoggerPath  = flag.String("logger_path", "", "Set Logger Root Path.")
)

func initApplication(application *kelvins.Application) error {
	flag.Parse()
	kelvins.ServerName = application.Name

	rootPath := DefaultLoggerRootPath
	if application.LoggerRootPath != "" {
		rootPath = application.LoggerRootPath
	}
	if *flagLoggerPath != "" {
		rootPath = *flagLoggerPath
	}
	loggerLevel := DefaultLoggerLevel
	if application.LoggerLevel != "" {
		loggerLevel = application.LoggerLevel
	}
	if *flagLoggerLevel != "" {
		loggerLevel = *flagLoggerLevel
	}

	err := log.InitGlobalConfig(rootPath, loggerLevel, application.Name)
	if err != nil {
		return fmt.Errorf("log.InitGlobalConfig: %v", err)
	}

	return nil
}

// setupCommonVars setup application global vars.
func setupCommonVars(application *kelvins.Application) error {
	var err error

	if kelvins.MysqlSetting != nil && kelvins.MysqlSetting.Host != "" {
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

	if kelvins.ServerSetting != nil && kelvins.ServerSetting.PIDFile != "" {
		kelvins.PIDFile = kelvins.ServerSetting.PIDFile
	} else {
		wd, _ := os.Getwd()
		kelvins.PIDFile = fmt.Sprintf("%s/%s.pid", wd, kelvins.ServerName)
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

func appShutdown(application *kelvins.Application) error {
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
	time.AfterFunc(30*time.Second, func() {
		logging.Infof("App Graceful shutdown timed out")
		os.Exit(1)
	})
}
