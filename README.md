# kelvins
[![kelvins](logo.png)](https://gitee.com/kelvins-io)   

go/golang微服务框架

### 支持特性
注册服务，发现服务，grpc/http gateway，cron，queue，http/gin服务，插拔式配置加载，双orm支持，mysql，mongo支持，事件总线，日志，异步任务池，   
Prometheus/pprof监控，进程优雅重启，应用自定义配置，启动flag参数指定，应用hook，工具类（由kelvins-io/common支持），全局变量vars，   
在线应用负载均衡，启动命令，RPC健康检查，接入授权，ghz压力测试tool

#### 即将支持
限流，熔断，异常接入sentry，kelvins-tools工具箱（一键生成应用，运维部署等）

### 软件环境
> go 1.13.15

### 运行环境变量
etcd集群地址   
ETCDV3_SERVER_URLS     
```
笔者自己环境的配置（仅做参考） 
export ETCDCTL_API=3
export ETCDV3_SERVER_URL=http://10.211.55.4:2379,http://10.211.55.7:2379
如果自己搭建etcd集群最少需要两个节点（一主一从）本地搭建参考：https://gitee.com/cristiane/micro-mall-api/blob/master/%E5%BE%AE%E5%95%86%E5%9F%8EETCD%E9%83%A8%E7%BD%B2.pdf
```

~~GO_ENV~~
```
运行环境标识，可选值有：dev，test，release，prod；分别对应开发环境，测试环境，预发布/灰度环境，prod正式环境，本地配置export GO_ENV=dev就好
```
新版本的kelvins不再依赖GO_ENV   

### 目前最新版支持的配置文件
``` 
配置文件默认就在项目根目录下/etc/app.ini文件，大多数micro-mall-开头的项目已经包含了所需的配置文件，根据自己本地的配置修改即可
```
截止最新版kelvins，全部支持的项目配置（./etc/app.ini）内容   
除了必要的配置项外，其余配置都是你不配置kelvins就不会去加载初始&当然对应的功能也不能用   
服务配置：端口，传输协议等   

**必要配置项：   
kelvins-server    
```ini
[kelvins-server]
IsRecordCallResponse = true
Network = "tcp"
Environment = "dev"
PIDFile = "./kelvins-app.pid"
ReadTimeout = 30 #秒
WriteTimeout = 30 #秒
IdleTimeout = 30 #秒
```
日志：级别，路径等   
kelvins-logger   
```ini
[kelvins-logger]
RootPath = "./logs"
Level = "debug"
```

--自选配置项：   
MySQL：连接配置信息   
kelvins-mysql   
```ini
[kelvins-mysql]
Host = "127.0.0.1:3306"
UserName = "root"
Password = "2123afsdfadsffasdf"
DBName = "micro_mall_user"
Charset = "utf8"
PoolNum =  10
MaxIdleConns = 5
ConnMaxLifeSecond = 3600
MultiStatements = true
ParseTime = true
```
Redis：连接配置信息   
kelvins-redis   
```ini
[kelvins-redis]
Host = "127.0.0.1:6379"
Password = "f434rtafadsfasd"
DB = 1
PoolNum = 10
```
MongoDB：连接配置信息   
kelvins-mongodb   
```ini
[mongodb-config]
Uri = "mongodb://127.0.0.1:27017"
Username = "admin"
Password = "fadfadsf3"
Database = "micro_mall_sku"
AuthSource = "admin"
MaxPoolSize = 9
MinPoolSize = 3
```
队列功能需要的Redis配置   
kelvins-queue-redis   
```ini
[kelvins-queue-redis]
Broker = "redis://xxx"
DefaultQueue = "user_register_notice"
ResultBackend = "redis://fdfsfds@127.0.0.1:6379/8"
ResultsExpireIn = 3600
```
队列功能-阿里云队列（在阿里云购买的amqp）   
kelvins-queue-ali-amqp   
```ini
[kelvins-queue-ali-amqp]
AccessKey = "ffwefwettgt"
SecretKey = "dfadfasdfasd"
AliUid = 11
EndPoint = "localhost:0909"
VHost = "/kelvins-io"
DefaultQueue = "queue1"
ResultBackend = "redis://xxx@127.0.0.1:6379/8"
ResultsExpireIn = 3600
Exchange = "user_register_notice"
ExchangeType = "direct"
BindingKey = "user_register_notice"
PrefetchCount = 6
```
队列功能-amqp协议（也就是自己搭建的rabbitmq）   
kelvins-queue-amqp   
```ini
[queue-user-register-notice]
Broker = "amqp://micro-mall:xx@127.0.0.1:5672/micro-mall"
DefaultQueue = "user_register_notice"
ResultBackend = "redis://xxx@127.0.0.1:6379/8"
ResultsExpireIn = 3600
Exchange = "user_register_notice"
ExchangeType = "direct"
BindingKey = "user_register_notice"
PrefetchCount = 5
TaskRetryCount = 3
TaskRetryTimeout = 3600
```
队列功能，事件订阅功能（阿里云）   
kelvins-queue-ali-rocketmq   
```ini
[kelvins-queue-ali-rocketmq]
BusinessName = "kelvins-io"
RegionId = "firuutu"
AccessKey = "dqwkjd8njf"
SecretKey = "xoik-94m3"
InstanceId = "8fdac-90jcc"
HttpEndpoint = "https://aliyun.com"
```
队列消费者配置   
kelvins-queue-server   
```ini
[kelvins-queue-server]
WorkerConcurrency = 5
CustomQueueList = "queue1,queue2"
```
异步任务池   
kelvins-gpool   
```ini
[kelvins-gpool]
WorkerNum = 10
JobChanLen = 1000
```
RPC接入授权
kelvins-auth
```ini
[kelvins-auth]
Token = "abc1234"
TransportSecurity = false
```

++自定义配置项，根据项目本身而定    
micro-mall-api/etc/app.ini#EmailConfig就属于自定义配置项    

--启动flag参数   
说明：flag参数优先级高于配置文件中同名配置参数，flag参数均可不指定，默认从进程运行目录/etc/app.ini加载，日志文件路径默认在进程运行目录/logs   
-logger_level 日志级别   
-logger_path  日志文件路径   
-conf_file  配置文件（ini文件）路径  
-env 运行环境变量：dev test prod    
-s start 启动进程   
-s restart 重启当前进程   
-s stop 停止当前进程   

### APP注册参考
请在你的应用main.go中注册application
```go
package main

import (
	"crypto/tls"
	"gitee.com/cristiane/micro-mall-pay/startup"
	"gitee.com/kelvins-io/kelvins"
	"gitee.com/kelvins-io/kelvins/app"
)

const APP_NAME = "micro-mall-pay"

func main() {
	application := &kelvins.GRPCApplication{
		Application: &kelvins.Application{
			LoadConfig: startup.LoadConfig,
			SetupVars:  startup.SetupVars,
			Name:       APP_NAME,
		},
		TlsConfig: &tls.Config{
			// 配置应用证书，仅仅对grpc,http类应用支持
		},
		NumServerWorkers:     200, // rpc工作协程数
		RegisterHealthServer: startup.RegisterGRPCHealthCheck, // 异步RPC健康检查
		RegisterGRPCServer: startup.RegisterGRPCServer,
		RegisterGateway:    startup.RegisterGateway, // RPC gateway接入
		RegisterHttpRoute:  startup.RegisterHttpRoute, // HTTP mutex
	}
	app.RunGRPCApplication(application)
}
```

### 更新日志
时间 | 内容 |  贡献者 | 备注  
---|------|------|---
2020-6-1 | 基础开发阶段 | https://gitee.com/cristiane | 结构思考，基础搭建
2020-8-27 | 预览版上线 | https://gitee.com/cristiane | 支持gRPC，HTTP，crontab，queue类应用
2020-9-10 | 增加MongoDB支持 | https://gitee.com/cristiane | 基于配置加载MongoDB来初始化应用
2020-9-13 | 支持Redis队列 | https://gitee.com/cristiane | 基于配置加载queue-Redis来初始化应用
2020-11-24 | v2 | https://gitee.com/cristiane | 若干更新
2021-4-5 | 支持应用优雅重启，退出 | https://gitee.com/cristiane | 基于操作系统信号，各平台有差异
2021-4-19 | 支持gin | https://gitee.com/cristiane | 允许将gin http handler注册到应用
2021-7-9 | 兼容Windows | https://gitee.com/cristiane | 修复Windows平台应用不能启动问题
2021-8-1 | 应用退出执行函数优化 | https://gitee.com/cristiane | 应用退出时异常处理
2021-8-1 | 应用支持负载均衡 | https://gitee.com/cristiane | 针对gRPC，http应用；同一应用多实例自动负载均衡
2021-8-7 | 启动命令 | https://gitee.com/cristiane | -s启动参数，支持启动进程，重启进程，停止进程
2021-8-13 | RPC健康检查 | https://gitee.com/cristiane | 支持使用grpc-health-probe等工具进行健康检查
2021-8-14 | RPC接入授权-token | https://gitee.com/cristiane | RPC应用支持开启接入授权
2021-8-14 | RPC-ghz压测试工具 | https://gitee.com/cristiane | 支持对RPC应用进行压力测试并输出报告

### 业务应用
micro-mall-api系列共计16个服务：https://gitee.com/cristiane/micro-mall-api

###技术交流
QQ群：578859618   
![avatar](./交流群.JPG)   
邮件：1225807604@qq.com