package kelvins

import (
	"gitee.com/kelvins-io/common/log"
	"gitee.com/kelvins-io/kelvins/config/setting"
	"gitee.com/kelvins-io/kelvins/util/goroutine"
	"github.com/gomodule/redigo/redis"
	"github.com/jinzhu/gorm"
	"github.com/qiniu/qmgo"
	"xorm.io/xorm"
)

// this VARS user should only read

// RedisConn is a global vars for redis connect，close by Framework exit
var RedisConn *redis.Pool

// GORM_DBEngine is a global vars for mysql connect，close by Framework exit
var GORM_DBEngine *gorm.DB

// XORM_DBEngine is a global vars for mysql connect，close by Framework exit
var XORM_DBEngine xorm.EngineInterface

// FrameworkLogger is a global var for Framework log
var FrameworkLogger log.LoggerContextIface

// ErrLogger is a global vars for application to log err msg.
var ErrLogger log.LoggerContextIface

// AccessLogger is a global vars for application to log access log
var AccessLogger log.LoggerContextIface

// BusinessLogger is a global vars for application to log business log
var BusinessLogger log.LoggerContextIface

// LoggerSetting is maps config section "kelvins-logger"
var LoggerSetting *setting.LoggerSettingS

// ServerSetting is maps config section "kelvins-server"
var ServerSetting *setting.ServerSettingS

// RPCServerParamsSetting is maps config section "kelvins-rpc-server"
var RPCServerParamsSetting *setting.RPCServerParamsS

// RPCAuthSetting is maps config section "kelvins-auth"
var RPCAuthSetting *setting.RPCAuthSettingS

// RPCServerKeepaliveParamsSetting is maps config section "kelvins-rpc-server-kp"
var RPCServerKeepaliveParamsSetting *setting.RPCServerKeepaliveParamsS

// RPCServerKeepaliveEnforcementPolicySetting is maps config section "kelvins-rpc-server-kep"
var RPCServerKeepaliveEnforcementPolicySetting *setting.RPCServerKeepaliveEnforcementPolicyS

// RPCClientKeepaliveParamsSetting is maps config section "kelvins-rpc-client-kp"
var RPCClientKeepaliveParamsSetting *setting.RPCClientKeepaliveParamsS

// RPCTransportBufferSetting is maps config section "kelvins-rpc-transport-buffer"
var RPCTransportBufferSetting *setting.RPCTransportBufferS

// MysqlSetting is maps config section "kelvins-mysql"
var MysqlSetting *setting.MysqlSettingS

// RedisSetting is maps config section "kelvins-redis"
var RedisSetting *setting.RedisSettingS

// QueueRedisSetting is maps config section "kelvins-queue-redis"
var QueueRedisSetting *setting.QueueRedisSettingS

// QueueServerSetting is maps config section "kelvins-queue-server"
var QueueServerSetting *setting.QueueServerSettingS

// QueueAliAMQPSetting is maps config section "kelvins-queue-amqp"
var QueueAliAMQPSetting *setting.QueueAliAMQPSettingS

// AliRocketMQSetting is maps config section "kelvins-queue-ali-rocketmq"
var AliRocketMQSetting *setting.AliRocketMQSettingS

// QueueAMQPSetting is maps config section "kelvins-queue-amqp"
var QueueAMQPSetting *setting.QueueAMQPSettingS

// MongoDBSetting is maps config section "kelvins-mongodb"
var MongoDBSetting *setting.MongoDBSettingS

// MongoDBClient is qmgo-client for mongodb，close by Framework exit
var MongoDBClient *qmgo.QmgoClient

// GPoolSetting is maps config section "kelvins-gpool"
var GPoolSetting *setting.GPoolSettingS

// GPool is goroutine pool，close by Framework exit
var GPool *goroutine.Pool

// PIDFile is process pid，manage by Framework user only read
var PIDFile string

// ServerName is server name
var ServerName string

// AppCloseCh is app shutdown notice，close by Framework exit; user only read
var AppCloseCh <-chan struct{}
