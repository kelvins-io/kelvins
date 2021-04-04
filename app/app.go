package app

import (
	"flag"
	"fmt"
	"gitee.com/kelvins-io/common/log"
	"gitee.com/kelvins-io/kelvins"
	"gitee.com/kelvins-io/kelvins/setup"
	goroute "gitee.com/kelvins-io/kelvins/util/groutine"
	"os"
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
	kelvins.ServerName = application.Name

	flag.Parse()
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
		kelvins.GPool = goroute.NewPool(kelvins.GPoolSetting.WorkerNum, kelvins.GPoolSetting.JobChanLen)
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
