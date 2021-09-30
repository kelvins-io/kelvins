package kelvins

import (
	"gitee.com/kelvins-io/common/event"
	"gitee.com/kelvins-io/common/log"
	"gitee.com/kelvins-io/common/queue"
	"gitee.com/kelvins-io/kelvins/config/setting"
	"gitee.com/kelvins-io/kelvins/util/goroutine"
	"github.com/gomodule/redigo/redis"
	"github.com/jinzhu/gorm"
	"github.com/qiniu/qmgo"
	"xorm.io/xorm"
)

// this VARS user should only read

// RedisConn is a global vars for redis connect，close by Framework exit May be nil
var RedisConn *redis.Pool

// GORM_DBEngine is a global vars for mysql connect，close by Framework exit May be nil
var GORM_DBEngine *gorm.DB

// XORM_DBEngine is a global vars for mysql connect，close by Framework exit May be nil
var XORM_DBEngine xorm.EngineInterface

// FrameworkLogger is a global var for Framework log
var FrameworkLogger log.LoggerContextIface

// ErrLogger is a global vars for application to log err msg.
var ErrLogger log.LoggerContextIface

// AccessLogger is a global vars for application to log access log
var AccessLogger log.LoggerContextIface

// BusinessLogger is a global vars for application to log business log
var BusinessLogger log.LoggerContextIface

// LoggerSetting is maps config section "kelvins-logger" May be nil
var LoggerSetting *setting.LoggerSettingS

// ServerSetting is maps config section "kelvins-server" May be nil
var ServerSetting *setting.ServerSettingS

// HttpServerSetting is maps config section "kelvins-http-server" May be nil
var HttpServerSetting *setting.HttpServerSettingS

// HttpRateLimitSetting is maps config section "kelvins-http-rate-limit" may be nil
var HttpRateLimitSetting *setting.HttpRateLimitSettingS

// JwtSetting is maps config section "kelvins-jwt" may be nil
var JwtSetting *setting.JwtSettingS

// RPCServerParamsSetting is maps config section "kelvins-rpc-server" May be nil
var RPCServerParamsSetting *setting.RPCServerParamsS

// RPCAuthSetting is maps config section "kelvins-rpc-auth" May be nil
var RPCAuthSetting *setting.RPCAuthSettingS

// RPCRateLimitSetting is maps config section "kelvins-rpc-rate-limit" may be nil
var RPCRateLimitSetting *setting.RPCRateLimitSettingS

// RPCServerKeepaliveParamsSetting is maps config section "kelvins-rpc-server-kp" May be nil
var RPCServerKeepaliveParamsSetting *setting.RPCServerKeepaliveParamsS

// RPCServerKeepaliveEnforcementPolicySetting is maps config section "kelvins-rpc-server-kep" May be nil
var RPCServerKeepaliveEnforcementPolicySetting *setting.RPCServerKeepaliveEnforcementPolicyS

// RPCClientKeepaliveParamsSetting is maps config section "kelvins-rpc-client-kp" May be nil
var RPCClientKeepaliveParamsSetting *setting.RPCClientKeepaliveParamsS

// RPCTransportBufferSetting is maps config section "kelvins-rpc-transport-buffer" May be nil
var RPCTransportBufferSetting *setting.RPCTransportBufferS

// MysqlSetting is maps config section "kelvins-mysql" May be nil
var MysqlSetting *setting.MysqlSettingS

// RedisSetting is maps config section "kelvins-redis" May be nil
var RedisSetting *setting.RedisSettingS

// QueueRedisSetting is maps config section "kelvins-queue-redis" May be nil
var QueueRedisSetting *setting.QueueRedisSettingS

// QueueServerSetting is maps config section "kelvins-queue-server" May be nil
var QueueServerSetting *setting.QueueServerSettingS

// QueueAliAMQPSetting is maps config section "kelvins-queue-amqp" May be nil
var QueueAliAMQPSetting *setting.QueueAliAMQPSettingS

// AliRocketMQSetting is maps config section "kelvins-queue-ali-rocketmq" May be nil
var AliRocketMQSetting *setting.AliRocketMQSettingS

// QueueAMQPSetting is maps config section "kelvins-queue-ali-amqp" May be nil
var QueueAMQPSetting *setting.QueueAMQPSettingS

// QueueServerRedis is maps config section "kelvins-queue-redis" May be nil
var QueueServerRedis *queue.MachineryQueue

// QueueServerAMQP is maps config section "kelvins-queue-amqp" May be nil
var QueueServerAMQP *queue.MachineryQueue

// QueueServerAliAMQP is maps config section "kelvins-queue-ali-amqp" May be nil
var QueueServerAliAMQP *queue.MachineryQueue

// EventServerAliRocketMQ is maps config section "kelvins-queue-ali-rocketmq" May be nil
var EventServerAliRocketMQ *event.EventServer

// MongoDBSetting is maps config section "kelvins-mongodb" May be nil
var MongoDBSetting *setting.MongoDBSettingS

// MongoDBClient is qmgo-client for mongodb，close by Framework exit May be nil
var MongoDBClient *qmgo.QmgoClient

// GPoolSetting is maps config section "kelvins-gpool" May be nil
var GPoolSetting *setting.GPoolSettingS

// GPool is goroutine pool，close by Framework exit May be nil
var GPool *goroutine.Pool

// PIDFile is process pid，manage by Framework user only read
var PIDFile string

// AppName is app name
var AppName string

// AppCloseCh is app shutdown notice，close by Framework exit; user only read
var AppCloseCh <-chan struct{}

// GRPCAppInstance is *GRPCApplication instance May be nil
var GRPCAppInstance *GRPCApplication

// CronAppInstance is *CronApplication instance May be nil
var CronAppInstance *CronApplication

// QueueAppInstance is **QueueApplication instance May be nil
var QueueAppInstance *QueueApplication

// HttpAppInstance is *HTTPApplication instance May be nil
var HttpAppInstance *HTTPApplication
