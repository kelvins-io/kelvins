package setting

import (
	"gitee.com/kelvins-io/common/log"
	"time"
)

// ServerSettingS defines for server.
type ServerSettingS struct {
	AppName     string
	PIDFile     string
	Environment string
}

func (s *HttpServerSettingS) GetReadTimeout() time.Duration {
	return time.Duration(s.ReadTimeout) * time.Second
}

func (s *HttpServerSettingS) GetWriteTimeout() time.Duration {
	return time.Duration(s.WriteTimeout) * time.Second
}

func (s *HttpServerSettingS) GetIdleTimeout() time.Duration {
	return time.Duration(s.IdleTimeout) * time.Second
}

func (s *HttpServerSettingS) SetAddr(addr string) {
	s.addr = addr
}

func (s *HttpServerSettingS) GetAddr() string {
	return s.addr
}

type HttpServerSettingS struct {
	Network      string
	ReadTimeout  int
	WriteTimeout int
	IdleTimeout  int
	SupportH2    bool
	addr         string
}

type JwtSettingS struct {
	Secret            string
	TokenExpireSecond int
}

type RPCServerParamsS struct {
	NumServerWorkers             int64
	ConnectionTimeout            int64 // unit second
	DisableClientDialHealthCheck bool
	DisableHealthServer          bool
}

type RPCAuthSettingS struct {
	Token             string
	ExpireSecond      int
	TransportSecurity bool
}

type RPCServerKeepaliveParamsS struct {
	PingClientIntervalTime int64
	MaxConnectionIdle      int64
}

type RPCServerKeepaliveEnforcementPolicyS struct {
	ClientMinIntervalTime int64
	PermitWithoutStream   bool
}

type RPCClientKeepaliveParamsS struct {
	PingServerIntervalTime int64
	PermitWithoutStream    bool
}

type RPCTransportBufferS struct {
	ServerReadBufSizeKB  int
	ServerWriteBufSizeKB int
	ClientReadBufSizeKB  int
	ClientWriteBufSizeKB int
}

type LoggerSettingS struct {
	RootPath string
	Level    string
}

// MysqlSettingS defines for connecting mysql.
type MysqlSettingS struct {
	Host              string
	UserName          string
	Password          string
	DBName            string
	Charset           string
	MaxIdle           int
	MaxOpen           int
	Loc               string
	ConnMaxLifeSecond int
	MultiStatements   bool
	ParseTime         bool
	ConnectionTimeout string // time unit eg: 2h 3s
	WriteTimeout      string // time unit eg: 2h 3s
	ReadTimeout       string // time unit eg: 2h 3s
	// only app use
	LoggerLevel string
	Environment string
	Logger      log.LoggerContextIface
}

// RedisSettingS defines for connecting redis.
type RedisSettingS struct {
	Host           string
	Password       string
	MaxIdle        int
	MaxActive      int
	IdleTimeout    int // unit second
	ConnectTimeout int // unit second
	ReadTimeout    int // unit second
	WriteTimeout   int // unit second
	DB             int
}

// QueueServerSettingS defines what queue server needs.
type QueueServerSettingS struct {
	WorkerConcurrency int
	CustomQueueList   []string
}

// QueueRedisSettingS defines for redis queue.
type QueueRedisSettingS struct {
	Broker           string
	DefaultQueue     string
	ResultBackend    string
	ResultsExpireIn  int
	DisableConsume   bool
	TaskRetryCount   int
	TaskRetryTimeout int
}

// QueueAliAMQPSettingS defines for ali yun AMQP queue
type QueueAliAMQPSettingS struct {
	AccessKey        string
	SecretKey        string
	AliUid           int
	EndPoint         string
	VHost            string
	DefaultQueue     string
	ResultBackend    string
	ResultsExpireIn  int
	Exchange         string
	ExchangeType     string
	BindingKey       string
	PrefetchCount    int
	TaskRetryCount   int
	TaskRetryTimeout int
	DisableConsume   bool
}

type QueueAMQPSettingS struct {
	Broker           string
	DefaultQueue     string
	ResultBackend    string
	ResultsExpireIn  int
	Exchange         string
	ExchangeType     string
	BindingKey       string
	PrefetchCount    int
	TaskRetryCount   int
	TaskRetryTimeout int
	DisableConsume   bool
}

// AliRocketMQSettingS defines for ali yun RocketMQ queue
type AliRocketMQSettingS struct {
	BusinessName string
	RegionId     string
	AccessKey    string
	SecretKey    string
	InstanceId   string
	HttpEndpoint string
}

type MongoDBSettingS struct {
	Uri         string
	Username    string
	Password    string
	Database    string
	AuthSource  string
	MaxPoolSize int
	MinPoolSize int
}

type GPoolSettingS struct {
	WorkerNum  int
	JobChanLen int
}
