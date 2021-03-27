package config

import (
	"gitee.com/kelvins-io/kelvins"
	"gitee.com/kelvins-io/kelvins/config/setting"
	"gopkg.in/ini.v1"
	"log"
)

const (
	// ConfFileName defines config file name.
	ConfFileName = "./etc/app.ini"
	// SectionServer is a section name for grpc server.
	SectionServer = "kelvins-server"
	// SectionLogger is a section name for logger.
	SectionLogger = "kelvins-logger"
	// SectionMysql is a sectoin name for mysql.
	SectionMysql = "kelvins-mysql"
	// SectionRedis is a section name for redis.
	SectionRedis = "kelvins-redis"
	// SectionMongodb is a section name for mongodb
	SectionMongoDB = "kelvins-mongodb"
	// SectionQueueRedis is a section name for redis queue
	SectionQueueRedis = "kelvins-queue-redis"
	// SectionQueueAliAMQP is a section name for aliamqp
	SectionQueueAliAMQP = "kelvins-queue-ali-amqp"
	// SectionQueueAMQP is a section name for amqp
	SectionQueueAMQP = "kelvins-queue-amqp"
	// SectionQueueAliRocketMQ is a section name for ali-rocketmq
	SectionQueueAliRocketMQ = "kelvins-queue-ali-rocketmq"
	// SectionQueueServer is a section name for queue-server
	SectionQueueServer = "kelvins-queue-server"
)

// cfg reads file app.ini.
var cfg *ini.File

// LoadDefaultConfig loads config form cfg.
func LoadDefaultConfig(application *kelvins.Application) error {
	// Setup cfg object
	var err error
	cfg, err = ini.Load(ConfFileName)
	if err != nil {
		return err
	}

	// Setup default settings
	for _, sectionName := range cfg.SectionStrings() {
		if sectionName == SectionServer {
			kelvins.ServerSetting = new(setting.ServerSettingS)
			MapConfig(sectionName, kelvins.ServerSetting)
			continue
		}
		if sectionName == SectionLogger {
			kelvins.LoggerSetting = new(setting.LoggerSettingS)
			MapConfig(sectionName, kelvins.LoggerSetting)
			application.LoggerRootPath = kelvins.LoggerSetting.RootPath
			application.LoggerLevel = kelvins.LoggerSetting.Level
			continue
		}
		if sectionName == SectionMysql {
			kelvins.MysqlSetting = new(setting.MysqlSettingS)
			MapConfig(sectionName, kelvins.MysqlSetting)
			continue
		}
		if sectionName == SectionRedis {
			kelvins.RedisSetting = new(setting.RedisSettingS)
			MapConfig(sectionName, kelvins.RedisSetting)
			continue
		}
		if sectionName == SectionMongoDB {
			kelvins.MongoDBSetting = new(setting.MongoDBSettingS)
			MapConfig(sectionName, kelvins.MongoDBSetting)
			continue
		}
		if sectionName == SectionQueueRedis {
			kelvins.QueueRedisSetting = new(setting.QueueRedisSettingS)
			MapConfig(sectionName, kelvins.QueueRedisSetting)
			continue
		}
		if sectionName == SectionQueueAliAMQP {
			kelvins.QueueAliAMQPSetting = new(setting.QueueAliAMQPSettingS)
			MapConfig(sectionName, kelvins.QueueAliAMQPSetting)
			continue
		}
		if sectionName == SectionQueueAMQP {
			kelvins.QueueAMQPSetting = new(setting.QueueAMQPSettingS)
			MapConfig(sectionName, kelvins.QueueAMQPSetting)
			continue
		}
		if sectionName == SectionQueueAliRocketMQ {
			kelvins.AliRocketMQSetting = new(setting.AliRocketMQSettingS)
			MapConfig(sectionName, kelvins.AliRocketMQSetting)
			continue
		}
		if sectionName == SectionQueueServer {
			kelvins.QueueServerSetting = new(setting.QueueServerSettingS)
			MapConfig(sectionName, kelvins.QueueServerSetting)
			continue
		}
	}
	return nil
}

// MapConfig uses cfg to map config.
func MapConfig(section string, v interface{}) {
	log.Printf("[info] Load default config %s", section)
	sec, err := cfg.GetSection(section)
	if err != nil {
		log.Fatalf("[err] Fail to parse '%s': %v", section, err)
	}
	err = sec.MapTo(v)
	if err != nil {
		log.Fatalf("[err] %s section map to setting err: %v", section, err)
	}
}
