package app

import (
	"flag"
	"fmt"
	"gitee.com/kelvins-io/common/log"
	"gitee.com/kelvins-io/kelvins"
	"gitee.com/kelvins-io/kelvins/setup"
)

var (
	port       = flag.Int64("p", 0, "Set server port.")
	loggerPath = flag.String("logger_path", "", "Set Logger Root Path.")
)

// initApplication initalizes application.
func initApplication(application *kelvins.Application) error {
	const DefaultLoggerRootPath = "./logs"
	const DefaultLoggerLevel = "debug"

	rootPath := DefaultLoggerRootPath
	if application.LoggerRootPath != "" {
		rootPath = application.LoggerRootPath
	}
	loggerLevel := DefaultLoggerLevel
	if application.LoggerLevel != "" {
		loggerLevel = application.LoggerLevel
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
