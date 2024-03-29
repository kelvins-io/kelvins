package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gitee.com/kelvins-io/common/event"
	"gitee.com/kelvins-io/common/log"
	"gitee.com/kelvins-io/kelvins"
	"gitee.com/kelvins-io/kelvins/internal/config"
	"gitee.com/kelvins-io/kelvins/internal/logging"
	"gitee.com/kelvins-io/kelvins/internal/service/slb"
	"gitee.com/kelvins-io/kelvins/internal/service/slb/etcdconfig"
	"gitee.com/kelvins-io/kelvins/internal/util"
	"gitee.com/kelvins-io/kelvins/internal/vars"
	"gitee.com/kelvins-io/kelvins/setup"
	"gitee.com/kelvins-io/kelvins/util/goroutine"
	"gitee.com/kelvins-io/kelvins/util/startup"
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
	// 1 show app version
	showAppVersion(application)

	// 2. load app config
	err := config.LoadDefaultConfig(application)
	if err != nil {
		return err
	}

	// 3 setup app proc
	setupApplicationProcess(application)

	// 4 startup control
	next, err := startUpControl(kelvins.PIDFile)
	if err != nil {
		return err
	}
	if !next {
		close(appCloseCh)
		<-kelvins.AppCloseCh
		return nil
	}

	// 5 init user config
	if application.LoadConfig != nil {
		err = application.LoadConfig()
		if err != nil {
			return err
		}
	}

	// 6 init system vars
	kelvins.AppName = application.Name
	if kelvins.ServerSetting != nil && kelvins.ServerSetting.AppName != "" {
		kelvins.AppName = kelvins.ServerSetting.AppName
		application.Name = kelvins.ServerSetting.AppName
	}
	if kelvins.AppName == "" {
		logging.Fatal("Application name can not be empty")
	}

	// init logger environ vars
	flag.Parse()
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

	// init environment
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

	// 7 init log
	err = log.InitGlobalConfig(loggerPath, loggerLevel, application.Name)
	if err != nil {
		return fmt.Errorf("log.InitGlobalConfig: %v", err)
	}

	// 8. setup vars
	// setup app vars
	err = setupCommonVars(application)
	if err != nil {
		return err
	}
	// setup user vars
	if application.SetupVars != nil {
		err = application.SetupVars()
		if err != nil {
			return fmt.Errorf("application.SetupVars err: %v", err)
		}
	}
	return nil
}

// setup AppCloseCh pid
func setupApplicationProcess(application *kelvins.Application) {
	kelvins.AppCloseCh = appCloseCh
	vars.AppCloseCh = appCloseCh
	vars.Version = kelvins.Version
	if kelvins.ServerSetting != nil {
		if kelvins.ServerSetting.PIDFile != "" {
			kelvins.PIDFile = filepath.Dir(kelvins.ServerSetting.PIDFile)
		} else {
			wd, _ := os.Getwd()
			kelvins.PIDFile = fmt.Sprintf("%s/%s.pid", wd, application.Name)
			//if runtime.GOOS == "windows"  {
			//	kelvins.PIDFile = filepath.ToSlash(kelvins.PIDFile)
			//}
		}
	}
}

// setupCommonVars setup application global vars.
func setupCommonVars(application *kelvins.Application) error {
	var err error
	if kelvins.MysqlSetting != nil && kelvins.MysqlSetting.Host != "" {
		kelvins.MysqlSetting.LoggerLevel = application.LoggerLevel
		kelvins.MysqlSetting.Environment = application.Environment
		logger, err := log.GetCustomLogger("db-log", "mysql")
		if err != nil {
			return err
		}
		kelvins.MysqlSetting.Logger = logger
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

	if kelvins.G2CacheSetting != nil && kelvins.G2CacheSetting.RedisConfDSN != "" {
		kelvins.G2CacheEngine, err = setup.NewG2Cache(kelvins.G2CacheSetting, nil, nil)
		if err != nil {
			return err
		}
	}

	kelvins.FrameworkLogger, err = log.GetCustomLogger("framework", "framework")
	if err != nil {
		return err
	}
	vars.FrameworkLogger = kelvins.FrameworkLogger

	kelvins.ErrLogger, err = log.GetErrLogger("err")
	if err != nil {
		return err
	}
	vars.ErrLogger = kelvins.ErrLogger

	kelvins.BusinessLogger, err = log.GetBusinessLogger("business")
	if err != nil {
		return err
	}
	vars.BusinessLogger = kelvins.BusinessLogger

	kelvins.AccessLogger, err = log.GetAccessLogger("access")
	if err != nil {
		return err
	}
	vars.AccessLogger = kelvins.AccessLogger

	// init event server
	if kelvins.AliRocketMQSetting != nil && kelvins.AliRocketMQSetting.InstanceId != "" {
		// new event server
		var queueLogger log.LoggerContextIface
		if kelvins.ServerSetting != nil {
			switch kelvins.ServerSetting.Environment {
			case config.DefaultEnvironmentDev:
				queueLogger = kelvins.BusinessLogger
			case config.DefaultEnvironmentTest:
				queueLogger = kelvins.BusinessLogger
			default:
			}
		}
		eventServer, err := event.NewEventServer(&event.Config{
			BusinessName: kelvins.AliRocketMQSetting.BusinessName,
			RegionId:     kelvins.AliRocketMQSetting.RegionId,
			AccessKey:    kelvins.AliRocketMQSetting.AccessKey,
			SecretKey:    kelvins.AliRocketMQSetting.SecretKey,
			InstanceId:   kelvins.AliRocketMQSetting.InstanceId,
			HttpEndpoint: kelvins.AliRocketMQSetting.HttpEndpoint,
		}, queueLogger)
		if err != nil {
			return err
		}
		kelvins.EventServerAliRocketMQ = eventServer
		return nil
	}

	return nil
}

func setupCommonQueue(namedTaskFunc map[string]interface{}) error {
	if kelvins.QueueRedisSetting != nil && kelvins.QueueRedisSetting.Broker != "" {
		queueServ, err := setup.NewRedisQueue(kelvins.QueueRedisSetting, namedTaskFunc)
		if err != nil {
			return err
		}
		kelvins.QueueServerRedis = queueServ
	}
	if kelvins.QueueAMQPSetting != nil && kelvins.QueueAMQPSetting.Broker != "" {
		queueServ, err := setup.NewAMQPQueue(kelvins.QueueAMQPSetting, namedTaskFunc)
		if err != nil {
			return err
		}
		kelvins.QueueServerAMQP = queueServ
	}
	if kelvins.QueueAliAMQPSetting != nil && kelvins.QueueAliAMQPSetting.VHost != "" {
		queueServ, err := setup.NewAliAMQPQueue(kelvins.QueueAliAMQPSetting, namedTaskFunc)
		if err != nil {
			return err
		}
		kelvins.QueueServerAliAMQP = queueServ
	}

	return nil
}

// appCloseChOne is appCloseCh sync.Once
var appCloseChOne sync.Once
var appCloseCh = make(chan struct{})

func appShutdown(application *kelvins.Application) error {
	if !appProcessNext {
		return nil
	}
	appCloseChOne.Do(func() {
		close(appCloseCh)
	})
	if application.StopFunc != nil {
		err := application.StopFunc()
		if err != nil {
			return err
		}
	}
	if kelvins.GPool != nil {
		kelvins.GPool.Release()
		kelvins.GPool.WaitAll()
	}
	if kelvins.RedisConn != nil {
		err := kelvins.RedisConn.Close()
		if err != nil {
			return err
		}
	}
	if kelvins.GORM_DBEngine != nil {
		err := kelvins.GORM_DBEngine.Close()
		if err != nil {
			return err
		}
	}
	if kelvins.MongoDBClient != nil {
		err := kelvins.MongoDBClient.Close(context.Background())
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
	if !appProcessNext {
		return
	}
	time.AfterFunc(30*time.Second, func() {
		logging.Info("App Graceful shutdown timed out, force exit")
		os.Exit(1)
	})
}

var appProcessNext bool

func startUpControl(pidFile string) (next bool, err error) {
	next, err = startup.ParseCliCommand(pidFile)
	if next {
		appProcessNext = true
	}
	return
}

func showAppVersion(app *kelvins.Application) {
	var logo = `%20__%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20___%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%0A%2F%5C%20%5C%20%20%20%20%20%20%20%20%20%20%20%20%20%2F%5C_%20%5C%20%20%20%20%20%20%20%20%20%20%20%20%20__%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%20%0A%5C%20%5C%20%5C%2F'%5C%20%20%20%20%20%20%20__%5C%2F%2F%5C%20%5C%20%20%20%20__%20%20__%20%2F%5C_%5C%20%20%20%20%20___%20%20%20%20%20%20____%20%20%0A%20%5C%20%5C%20%2C%20%3C%20%20%20%20%20%2F'__%60%5C%5C%20%5C%20%5C%20%20%2F%5C%20%5C%2F%5C%20%5C%5C%2F%5C%20%5C%20%20%2F'%20_%20%60%5C%20%20%20%2F'%2C__%5C%20%0A%20%20%5C%20%5C%20%5C%5C%60%5C%20%20%2F%5C%20%20__%2F%20%5C_%5C%20%5C_%5C%20%5C%20%5C_%2F%20%7C%5C%20%5C%20%5C%20%2F%5C%20%5C%2F%5C%20%5C%20%2F%5C__%2C%20%60%5C%0A%20%20%20%5C%20%5C_%5C%20%5C_%5C%5C%20%5C____%5C%2F%5C____%5C%5C%20%5C___%2F%20%20%5C%20%5C_%5C%5C%20%5C_%5C%20%5C_%5C%5C%2F%5C____%2F%0A%20%20%20%20%5C%2F_%2F%5C%2F_%2F%20%5C%2F____%2F%5C%2F____%2F%20%5C%2F__%2F%20%20%20%20%5C%2F_%2F%20%5C%2F_%2F%5C%2F_%2F%20%5C%2F___%2F%20`
	var version = `[Major Version：%v Type：%v]`
	var remote = `┌───────────────────────────────────────────────────┐
│ [Gitee] https://gitee.com/kelvins-io/kelvins      │
│ [GitHub] https://github.com/kelvins-io/kelvins    │
└───────────────────────────────────────────────────┘`
	fmt.Println("based on")
	logoS, _ := url.QueryUnescape(logo)
	fmt.Println(logoS)
	fmt.Println("")
	fmt.Println(fmt.Sprintf(version, kelvins.Version, kelvins.AppTypeText[app.Type]))

	fmt.Println("")
	fmt.Println(remote)
	fmt.Println("Go Go Go ==>", app.Name)
}

func appRegisterEventProducer(register func(event.ProducerIface) error, appType int32) {
	server := kelvins.EventServerAliRocketMQ
	if server == nil {
		return
	}
	endPoint := server.GetEndpoint()
	// should not rely on the behavior of the application code
	producerFunc := func() {
		err := register(server)
		if err != nil {
			// kelvins.BusinessLogger must not be nil
			if kelvins.BusinessLogger != nil {
				kelvins.BusinessLogger.Errorf(context.Background(), "App(type:%v).EventServer endpoint(%v) RegisterEventProducer publish err: %v",
					kelvins.AppTypeText[appType], endPoint, err)
			}
			return
		}
	}
	if kelvins.GPool != nil {
		ok := kelvins.GPool.SendJobWithTimeout(producerFunc, 1*time.Second)
		if !ok {
			go producerFunc()
		}
	} else {
		go producerFunc()
	}
}

func appRegisterEventHandler(register func(event.EventServerIface) error, appType int32) {
	server := kelvins.EventServerAliRocketMQ
	if server == nil {
		return
	}
	endPoint := server.GetEndpoint()
	// should not rely on the behavior of the application code
	subscribeFunc := func() {
		err := register(server)
		if err != nil {
			// kelvins.BusinessLogger must not be nil
			if kelvins.BusinessLogger != nil {
				kelvins.BusinessLogger.Errorf(context.Background(), "App(type:%v).EventServer endpoint(%v) RegisterEventHandler subscribe err: %v",
					kelvins.AppTypeText[appType], endPoint, err)
			}
			return
		}
		// start event server
		err = server.Start()
		if err != nil {
			// kelvins.BusinessLogger must not be nil
			if kelvins.BusinessLogger != nil {
				kelvins.BusinessLogger.Errorf(context.Background(), "App(type:%v).EventServer endpoint(%v) Start consume err: %v",
					kelvins.AppTypeText[appType], endPoint, err)
			}
			return
		}
	}
	if kelvins.GPool != nil {
		ok := kelvins.GPool.SendJobWithTimeout(subscribeFunc, 1*time.Second)
		if !ok {
			go subscribeFunc()
		}
	} else {
		go subscribeFunc()
	}
}

func appUnRegisterServiceToEtcd(appName string, port int64) error {
	etcdServerUrls := config.GetEtcdV3ServerURLs()
	if etcdServerUrls == "" {
		if kelvins.ErrLogger != nil {
			kelvins.ErrLogger.Errorf(context.TODO(), "etcd not found environment variable(%v)", config.ENV_ETCDV3_SERVER_URLS)
		}
		return fmt.Errorf("etcd not found environment variable(%v)", config.ENV_ETCDV3_SERVER_URLS)
	}
	serviceLB := slb.NewService(etcdServerUrls, appName)
	serviceConfigClient := etcdconfig.NewServiceConfigClient(serviceLB)
	var registerSequence = getServiceSequence(serviceIP, strconv.Itoa(int(port)))
	err := serviceConfigClient.ClearConfig(registerSequence)
	if err != nil && err != etcdconfig.ErrServiceConfigKeyNotExist {
		if kelvins.ErrLogger != nil {
			kelvins.ErrLogger.Errorf(context.TODO(), "etcd serviceConfigClient ClearConfig err: %v, key: %v",
				err, serviceConfigClient.GetKeyName(appName, registerSequence))
		}
		return fmt.Errorf("etcd clear service port exception")
	}

	return nil
}

var serviceIP string

func appRegisterServiceToEtcd(serviceKind, appName string, initialPort int64) (int64, error) {
	var flagPort int64
	if initialPort > 0 { // use self define port to start process
		flagPort = initialPort
	} else {
		flagPort = int64(util.RandInt(50000, 60000))
	}
	currentPort := strconv.Itoa(int(flagPort))
	etcdServerUrls := config.GetEtcdV3ServerURLs()
	if etcdServerUrls == "" {
		if kelvins.ErrLogger != nil {
			kelvins.ErrLogger.Errorf(context.TODO(), "etcd not found environment variable(%v)", config.ENV_ETCDV3_SERVER_URLS)
		}
		return flagPort, fmt.Errorf("etcd not found environment variable(%v)", config.ENV_ETCDV3_SERVER_URLS)
	}
	var err error
	serviceIP, err = getOutBoundIP()
	if err != nil {
		return 0, fmt.Errorf("lookup out bound ip err(%v)", err)
	}

	var registerSequence = getServiceSequence(serviceIP, currentPort)
	serviceLB := slb.NewService(etcdServerUrls, appName)
	serviceConfigClient := etcdconfig.NewServiceConfigClient(serviceLB)
	serviceConfig, err := serviceConfigClient.GetConfig(registerSequence)
	if err != nil && err != etcdconfig.ErrServiceConfigKeyNotExist {
		if kelvins.ErrLogger != nil {
			kelvins.ErrLogger.Errorf(context.TODO(), "etcd serviceConfig.GetConfig err: %v ,sequence(%v)", err, registerSequence)
		}
		return flagPort, fmt.Errorf("etcd register service sequence(%v) exception", registerSequence)
	}
	if serviceConfig != nil {
		isExist := getServiceSequence(serviceConfig.ServiceIP, serviceConfig.ServicePort) == getServiceSequence(serviceIP, currentPort)
		if isExist {
			if kelvins.ErrLogger != nil {
				kelvins.ErrLogger.Errorf(context.TODO(), "etcd serviceConfig.GetConfig sequence(%v) exist", registerSequence)
			}
			return flagPort, fmt.Errorf("etcd register service sequence(%v) exist", registerSequence)
		}
	}

	err = serviceConfigClient.WriteConfig(registerSequence, etcdconfig.Config{
		ServiceVersion: kelvins.Version,
		ServicePort:    currentPort,
		ServiceIP:      serviceIP,
		ServiceKind:    serviceKind,
		LastModified:   time.Now().Format(kelvins.ResponseTimeLayout),
	})
	if err != nil {
		if kelvins.ErrLogger != nil {
			kelvins.ErrLogger.Errorf(context.TODO(), "etcd writeConfig err: %v，sequence(%v) ", err, currentPort)
		}
		err = fmt.Errorf("etcd register service port(%v) exception", currentPort)
	}
	vars.ServicePort = currentPort
	vars.ServiceIp = serviceIP
	return flagPort, err
}

func getServiceSequence(ip, port string) (key string) {
	ret := big.NewInt(0)
	ret.SetBytes(net.ParseIP(ip).To4())
	key = fmt.Sprintf("%v_%v", ret.Int64(), port)
	return
}

func getOutBoundIP() (ip string, err error) {
	conn, err := net.Dial("udp", "255.255.255.255:53")
	if err != nil {
		return
	}
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	ip = strings.Split(localAddr.String(), ":")[0]
	return
}

var (
	appInstanceOnce    int32
	appInstanceOnceErr = errors.New("the same app type can only be registered once")
)

func appInstanceOnceValidate() error {
	ok := atomic.CompareAndSwapInt32(&appInstanceOnce, 0, 1)
	if !ok {
		return appInstanceOnceErr
	}
	return nil
}
