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

// RedisConn is a global vars for redis connect.
var RedisConn *redis.Pool

// GORM_DBEngine is a global vars for mysql connect.
var GORM_DBEngine *gorm.DB

// XORM_DBEngine is a global vars for mysql connect.
var XORM_DBEngine xorm.EngineInterface

// FrameworkLogger is a global var for Framework log
var FrameworkLogger log.LoggerContextIface

// ErrLogger is a global vars for application to log err msg.
var ErrLogger log.LoggerContextIface

// AccessLogger is a global vars for application to log access log
var AccessLogger log.LoggerContextIface

// BusinessLogger is a global vars for application to log business log
var BusinessLogger log.LoggerContextIface

// LoggerSetting log setting
var LoggerSetting *setting.LoggerSettingS

// ServerSetting maps config section "kelvinsServer.*" from apollo.
var ServerSetting *setting.ServerSettingS

// MysqlSetting maps config section "kelvinsMysql.*" from apollo.
var MysqlSetting *setting.MysqlSettingS

// MysqlSetting maps config section "kelvinsRedis.*" from apollo.
var RedisSetting *setting.RedisSettingS

// QueueRedisSetting maps config section "kelvinsQueueRedis.*" from apollo.
var QueueRedisSetting *setting.QueueRedisSettingS

// QueueServerSetting maps config section "kelvinsQueueServer.*" from apollo.
var QueueServerSetting *setting.QueueServerSettingS

// QueueAliAMQPSetting maps config section "kelvinsQueueAliAMQP.*" from apollo.
var QueueAliAMQPSetting *setting.QueueAliAMQPSettingS

// AliRocketMQSetting
var AliRocketMQSetting *setting.AliRocketMQSettingS

// QueueAMQPSetting maps config section
var QueueAMQPSetting *setting.QueueAMQPSettingS

// MongoDBSetting maps config section mongodb.
var MongoDBSetting *setting.MongoDBSettingS

// MongoDBClient is qmgo-client for mongodb.
var MongoDBClient *qmgo.QmgoClient

// GPoolSetting is gpool setting
var GPoolSetting *setting.GPoolSettingS

// GPool is goroutine pool
var GPool *goroute.Pool

// PIDFile is process pid
var PIDFile string

// ServerName is server name
var ServerName string