package config

import (
	"flag"
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
	// SectionHttpServer is a section name for http
	SectionHttpServer = "kelvins-http-server"
	// SectionHttpRateLimit is a section mame for http
	SectionHttpRateLimit = "kelvins-http-rate-limit"
	// SectionLogger is a section name for logger.
	SectionLogger = "kelvins-logger"
	// SectionMysql is a sectoin name for mysql.
	SectionMysql = "kelvins-mysql"
	// SectionG2cache is a section name for g2cache
	SectionG2cache = "kelvins-g2cache"
	// SectionRedis is a section name for redis
	SectionRedis = "kelvins-redis"
	// SectionMongodb is a section name for mongodb
	SectionMongoDB = "kelvins-mongodb"
	// SectionQueueRedis is a section name for redis queue
	SectionQueueRedis = "kelvins-queue-redis"
	// SectionQueueAliAMQP is a section name for ali amqp
	SectionQueueAliAMQP = "kelvins-queue-ali-amqp"
	// SectionQueueAMQP is a section name for amqp
	SectionQueueAMQP = "kelvins-queue-amqp"
	// SectionQueueAliRocketMQ is a section name for ali-rocketmq
	SectionQueueAliRocketMQ = "kelvins-queue-ali-rocketmq"
	// SectionQueueServer is a section name for queue-server
	SectionQueueServer = "kelvins-queue-server"
	// SectionGPool is goroutine pool
	SectionGPool = "kelvins-gpool"
	// SectionJwt is jwt
	SectionJwt = "kelvins-jwt"
	// SectionAuth is rpc auth ,just for compatibility, it will be deleted in the future
	SectionAuth = "kelvins-auth"
	// SectionRPCAuth is rpc auth
	SectionRPCAuth = "kelvins-rpc-auth"
	// SectionRPCServerParams is server rpc params
	SectionRPCServerParams = "kelvins-rpc-server"
	// SectionRPCServerKeepaliveParams is server rpc keep alive params
	SectionRPCServerKeepaliveParams = "kelvins-rpc-server-kp"
	// SectionRPCServerKeepaliveEnforcementPolicy is server rpc keep alive enf policy
	SectionRPCServerKeepaliveEnforcementPolicy = "kelvins-rpc-server-kep"
	// SectionRPCClientKeepaliveParams is client rpc keep alive params
	SectionRPCClientKeepaliveParams = "kelvins-rpc-client-kp"
	// SectionRPCTransportBuffer is rpc transport buffer
	SectionRPCTransportBuffer = "kelvins-rpc-transport-buffer"
	// SectionRPCRateLimit is rpc rate limit
	SectionRPCRateLimit = "kelvins-rpc-rate-limit"
)

// cfg reads file app.ini.
var (
	cfg            *ini.File
	flagConfigPath = flag.String("conf_file", "", "set config file path")
)

// LoadDefaultConfig loads config form cfg.
func LoadDefaultConfig(application *kelvins.Application) error {
	flag.Parse()
	var configFile = ConfFileName
	if *flagConfigPath != "" {
		configFile = *flagConfigPath
	}

	// Setup cfg object
	var err error
	cfg, err = ini.Load(configFile)
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
		if sectionName == SectionHttpServer {
			kelvins.HttpServerSetting = new(setting.HttpServerSettingS)
			MapConfig(sectionName, kelvins.HttpServerSetting)
		}
		if sectionName == SectionHttpRateLimit {
			kelvins.HttpRateLimitSetting = new(setting.HttpRateLimitSettingS)
			MapConfig(sectionName, kelvins.HttpRateLimitSetting)
		}
		if sectionName == SectionJwt {
			kelvins.JwtSetting = new(setting.JwtSettingS)
			MapConfig(sectionName, kelvins.JwtSetting)
			continue
		}
		if sectionName == SectionRPCAuth || sectionName == SectionAuth {
			kelvins.RPCAuthSetting = new(setting.RPCAuthSettingS)
			MapConfig(sectionName, kelvins.RPCAuthSetting)
			continue
		}
		if sectionName == SectionRPCServerParams {
			kelvins.RPCServerParamsSetting = new(setting.RPCServerParamsS)
			MapConfig(sectionName, kelvins.RPCServerParamsSetting)
			continue
		}
		if sectionName == SectionRPCServerKeepaliveParams {
			kelvins.RPCServerKeepaliveParamsSetting = new(setting.RPCServerKeepaliveParamsS)
			MapConfig(sectionName, kelvins.RPCServerKeepaliveParamsSetting)
			continue
		}
		if sectionName == SectionRPCServerKeepaliveEnforcementPolicy {
			kelvins.RPCServerKeepaliveEnforcementPolicySetting = new(setting.RPCServerKeepaliveEnforcementPolicyS)
			MapConfig(sectionName, kelvins.RPCServerKeepaliveEnforcementPolicySetting)
			continue
		}
		if sectionName == SectionRPCClientKeepaliveParams {
			kelvins.RPCClientKeepaliveParamsSetting = new(setting.RPCClientKeepaliveParamsS)
			MapConfig(sectionName, kelvins.RPCClientKeepaliveParamsSetting)
			continue
		}
		if sectionName == SectionRPCTransportBuffer {
			kelvins.RPCTransportBufferSetting = new(setting.RPCTransportBufferS)
			MapConfig(sectionName, kelvins.RPCTransportBufferSetting)
			continue
		}
		if sectionName == SectionRPCRateLimit {
			kelvins.RPCRateLimitSetting = new(setting.RPCRateLimitSettingS)
			MapConfig(sectionName, kelvins.RPCRateLimitSetting)
			continue
		}
		if sectionName == SectionLogger {
			kelvins.LoggerSetting = new(setting.LoggerSettingS)
			MapConfig(sectionName, kelvins.LoggerSetting)
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
		if sectionName == SectionG2cache {
			kelvins.G2CacheSetting = new(setting.G2CacheSettingS)
			MapConfig(sectionName, kelvins.G2CacheSetting)
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
		if sectionName == SectionGPool {
			kelvins.GPoolSetting = new(setting.GPoolSettingS)
			MapConfig(sectionName, kelvins.GPoolSetting)
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
