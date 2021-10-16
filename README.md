# kelvins
[![kelvins](logo.png)](https://gitee.com/kelvins-io)   

go/golang微服务框架

### 支持特性
注册服务，发现服务，grpc/http gateway，cron，queue，http/gin服务（兼容h1.1，h2），插拔式配置加载，双orm支持，mysql，mongo支持，事件总线，日志，异步任务池，   
Prometheus/pprof监控，进程优雅重启，应用自定义配置，启动flag参数指定，应用hook，工具类（由kelvins-io/common支持），全局变量vars，   
在线应用负载均衡，启动命令，RPC健康检查，接入授权，ghz压力测试tool，gRPC服务端&客户端参数配置，在线服务限流，kelvins-tools工具箱，watch服务在线状态

#### 即将支持
熔断，异常接入sentry

### 软件环境
> go 1.13.15+

rpc采用gRPC，如果使用请安装依赖库
```shell
protoc 安装方法如下
wget https://github.com/google/protobuf/releases/download/v3.14.0/protobuf-all-3.14.0.zip
unzip protobuf-all-3.14.0.zip
cd protobuf-3.14.0/
./configure
make
make install
# 如果报错请执行
ldconfig
# grpc相关
go get -u google.golang.org/grpc@v1.32.0
go get -u google.golang.org/protobuf@v1.25.0
go get -u github.com/golang/protobuf/protoc-gen-go@v.1.4.3
go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway@v1.14.3
go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger
go get -u github.com/jteeuwen/go-bindata/...
go get -u github.com/elazarl/go-bindata-assetfs/...
python 2.7或3.5
```
### 运行环境变量
etcd集群地址   
ETCDV3_SERVER_URLS     
```
笔者自己环境的配置（仅做参考） 
export ETCDCTL_API=3
export ETCDV3_SERVER_URLS=http://10.211.55.4:2379,http://10.211.55.7:2379
如果自己搭建etcd集群最少需要两个节点（一主一从）本地搭建参考：https://gitee.com/cristiane/micro-mall-api/blob/master/%E5%BE%AE%E5%95%86%E5%9F%8EETCD%E9%83%A8%E7%BD%B2.pdf
```

~~GO_ENV~~
```
运行环境标识，可选值有：dev，test，release，prod；分别对应开发环境，测试环境，预发布/灰度环境，prod正式环境
```
新版本的kelvins不再依赖GO_ENV   

### 生成应用
参考kelvins-tools工具箱：https://gitee.com/kelvins-io/kelvins-tools/blob/master/README.md

### 目前最新版支持的配置文件
```
配置文件默认就在项目根目录下/etc/app.ini文件，大多数micro-mall-开头的项目已经包含了所需的配置文件，根据自己本地的配置修改即可
```
截止最新版kelvins，全部支持的项目配置（./etc/app.ini）内容   
配置项为空则使用默认值，当然相应业务对应的功能也不能用

**建议配置项：   
kelvins-server
Environment可选值：dev，test，release，prod   
AppName如果不为空则优先级高于代码注入的名字   
```ini
[kelvins-server]
AppName = "kelvins-template"
Environment = "dev"
PIDFile = "./kelvins-app.pid"
```

kelvins-logger   
日志：级别，路径等   
Level可选值：debug，warn，info，error
```ini
[kelvins-logger]
RootPath = "./logs"
Level = "debug"
```

--自选配置项：   
kelvins-http-server   
配置rpc，http服务   
 *Timeout时间单位#秒，Network和*Timeout仅对在线应用（h2c->gPRC，http）有效   
SupportH2仅对http服务有效，RPC调用默认就是H2   
http服务开启H2后任然兼容HTTP1.1，调用方想使用http2需要配置http客户端transport将协议升级到http2   
```ini
[kelvins-http-server]
Network = "tcp"
ReadTimeout = 30
WriteTimeout = 30
IdleTimeout = 30
SupportH2 = false
```

kelvins-http-rate-limit   
配置http服务限流   
MaxConcurrent 最大并发数（大于0有效）   
```ini
[kelvins-http-rate-limit]
MaxConcurrent = 0
```

kelvins-mysql   
MySQL：连接配置信息  
*Timeout 单位为时间表达字符串   
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
ConnectionTimeout = "30s"
WriteTimeout = "30s"
ReadTimeout = "30s"
```

kelvins-redis   
Redis：连接配置信息   
*Timeout 时间单位秒   
```ini
[kelvins-redis]
Host = "127.0.0.1:6379"
Password = "f434rtafadsfasd"
DB = 1
PoolNum = 10
IdleTimeout = 300
ConnectTimeout = 30
ReadTimeout = 30
WriteTimeout = 30
```

kelvins-mongodb   
MongoDB：连接配置信息   
```ini
[kelvins-mongodb]
Uri = "mongodb://127.0.0.1:27017"
Username = "admin"
Password = "fadfadsf3"
Database = "micro_mall_sku"
AuthSource = "admin"
MaxPoolSize = 9
MinPoolSize = 3
```

kelvins-queue-redis   
队列功能需要的Redis配置   
ResultsExpireIn 单位秒   
DisableConsume 表示作为queue类型应用时是否在该队列上执行消费任务   
```ini
[kelvins-queue-redis]
Broker = "redis://xxx"
DefaultQueue = "user_register_notice"
ResultBackend = "redis://fdfsfds@127.0.0.1:6379/8"
ResultsExpireIn = 3600
DisableConsume = false
TaskRetryCount = 3
TaskRetryTimeout = 60
```

kelvins-queue-ali-amqp   
队列功能-阿里云队列（在阿里云购买的amqp）   
ResultsExpireIn 单位秒   
PrefetchCount 一次获取消息数量   
DisableConsume 表示作为queue类型应用时是否在该队列上执行消费任务   
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
DisableConsume = false
TaskRetryCount = 3
TaskRetryTimeout = 60
```

kelvins-queue-amqp   
队列功能-amqp协议（也就是自己搭建的rabbitmq）   
ResultsExpireIn 单位秒   
PrefetchCount 一次获取消息数量   
DisableConsume 表示作为queue类型应用时是否在该队列上执行消费任务   
```ini
[kelvins-queue-amqp]
Broker = "amqp://micro-mall:xx@127.0.0.1:5672/micro-mall"
DefaultQueue = "user_register_notice"
ResultBackend = "redis://xxx@127.0.0.1:6379/8"
ResultsExpireIn = 3600
Exchange = "user_register_notice"
ExchangeType = "direct"
BindingKey = "user_register_notice"
PrefetchCount = 5
DisableConsume = false
TaskRetryCount = 3
TaskRetryTimeout = 60
```
 
kelvins-queue-ali-rocketmq   
队列功能，事件订阅功能（阿里云）  
```ini
[kelvins-queue-ali-rocketmq]
BusinessName = "kelvins-io"
RegionId = "firuutu"
AccessKey = "dqwkjd8njf"
SecretKey = "xoik-94m3"
InstanceId = "8fdac-90jcc"
HttpEndpoint = "https://aliyun.com"
```

kelvins-queue-server   
队列消费者配置   
WorkerConcurrency 表示启用多少个协程执行消费任务（默认值为任务列表长度）   
CustomQueueList 表示在配置的[kelvins-queue-amqp/ali-amqp/redis]队列上执行消费任务   
```ini
[kelvins-queue-server]
WorkerConcurrency = 5
CustomQueueList = "queue1,queue2,queue3"
```

kelvins-gpool   
异步任务池   
WorkerNum 异步协程的数量   
JobChanLen 等待任务队列长度上限     
```ini
[kelvins-gpool]
WorkerNum = 10
JobChanLen = 1000
```

kelvins-jwt   
gin中间件   
Secret 签名token用到的私钥   
TokenExpireSecond 签出的token有效期，单位秒   
```ini
[kelvins-jwt]
Secret = "私钥"
TokenExpireSecond = 3600
```

kelvins-auth   
RPC接入授权，不配置或者token为空表示不开启auth   
TransportSecurity 表示是否必须加密传输   
ExpireSecond token签名有效期，接收到请求的当前时间前后ExpireSecond秒都有效（默认30s）
推荐使用如下配置：   
```ini
[kelvins-rpc-auth]
Token = "abc1234"
ExpireSecond = 100
TransportSecurity = false
```

kelvins RPC-gRPC采用h2c（非TLS的http2） 接入方式（为了兼容http gateway）   
下面这些RPC参数（如无特殊无需配置）生效的优先级：配置文件 > 代码设置 > 默认值   
kelvins-rpc-server   
NumServerWorkers RPC服务端启用多少个常驻协程处理请求（为0表示每个请求一个协程处理）   
ConnectionTimeout 连接超时（单位秒，为0则使用默认值120s）（h2c接入rpc方式则无效）   
DisableHealthServer 为true表示当前服务不注册健康server（调用方调用时健康检查将无效）   
DisableClientDialHealthCheck 为true表示作为调用RPC服务的客户端不检查已建立的其它服务的rpc连接的健康状态   
RPC服务端参数，各参数为零则使用默认值   
```ini
[kelvins-rpc-server]
NumServerWorkers = 50
ConnectionTimeout = 120
DisableHealthServer = false
DisableClientDialHealthCheck = false
```

kelvins-rpc-rate-limit   
rpc服务限流   
MaxConcurrent 最大并发数（大于0有效）
```ini
[kelvins-rpc-rate-limit]
MaxConcurrent = 0
```

kelvins-rpc-server-kp   
RPC服务端keepalive参数   
PingClientIntervalTime 在这段时间后客户端没有任何活动服务器将主动ping客户端，单位秒   
MaxConnectionIdle 连接最大闲置时间，时间单位#秒   
```ini
[kelvins-rpc-server-kp]
PingClientIntervalTime = 3600
MaxConnectionIdle = 7200
```

kelvins-rpc-server-kep   
RPC服务端keepalive应对策略    
ClientMinIntervalTime 客户端keepalive的最小时间间隔，单位秒   
PermitWithoutStream 为true表示即使没有活动的RPC服务端也允许客户端发送心跳     
```ini
[kelvins-rpc-server-kep]
ClientMinIntervalTime = 300
PermitWithoutStream = true
```

kelvins-rpc-client-kp   
RPC客户端keepalive参数   
PingServerIntervalTime 客户端保活ping服务器的时间间隔，单位秒   
PermitWithoutStream 允许客户端在没有活动请求时也向服务器发送心跳   
```ini
[kelvins-rpc-client-kp]
PingServerIntervalTime = 360
PermitWithoutStream = true
```

kelvins-rpc-transport-buffer   
RPC传输buffer   
单位#KB   
```ini
[kelvins-rpc-transport-buffer]
ServerReadBufSizeKB = 32
ServerWriteBufSizeKB = 32
ClientReadBufSizeKB = 32
ClientWriteBufSizeKB = 32
```
++自定义配置项，根据项目本身而定    
micro-mall-api/etc/app.ini#EmailConfig就属于自定义配置项    

--启动flag参数   
说明：flag参数优先级高于配置文件中同名配置参数，flag参数均可不指定，默认从进程运行目录etc/app.ini加载，日志文件路径默认在进程运行目录logs   
-logger_level 日志级别   
-logger_path  日志文件路径   
-conf_file  配置文件（ini文件）路径  
-env 运行环境变量：dev test release prod    
-s start 启动进程   
-s restart 重启当前进程（Windows平台无效）   
-s stop 停止当前进程   

### 使用参考
1. 注册APP，在main.go中注册application
```go
package main

import (
	"crypto/tls"
	"gitee.com/cristiane/micro-mall-users/startup"
	"gitee.com/kelvins-io/kelvins"
	"gitee.com/kelvins-io/kelvins/app"
)

const APP_NAME = "micro-mall-users"

func main() {
	application := &kelvins.GRPCApplication{
		Application: &kelvins.Application{
			LoadConfig: startup.LoadConfig, // 加载自定义配置
			SetupVars:  startup.SetupVars, // 初始自定义变量
			Name:       APP_NAME,
		},
		TlsConfig: &tls.Config{
			// 配置应用证书，仅仅对grpc,http类应用支持
		},
		NumServerWorkers:     50, // rpc工作协程数
		RegisterGRPCHealthHandle: startup.RegisterGRPCHealthHandle, // 异步RPC健康状态维护
		RegisterGRPCServer: startup.RegisterGRPCServer, // 注册RPC
		RegisterGateway:    startup.RegisterGateway, // 注册gateway接入
		RegisterHttpRoute:  startup.RegisterHttpRoute, // 注册HTTP mutex
	}
	app.RunGRPCApplication(application) // 只能运行一个类型APP
}
```

2. RPC健康检查   
当RPC APP的 RegisterGRPCHealthHandle 不为nil且没有关闭health server时，kelvins就会为服务注入健康检查server，并在协程中启动监控维护函数   
使用grpc-health-probe工具命令进行健康检查   
kelvins rpc对健康检查接入做了免授权，所以即使服务开启了token验证也是可用的   
```shell
# 安装grpc-health-probe
git clone https://github.com/grpc-ecosystem/grpc-health-probe && cd grpc-health-probe && go build
# 查看命令
grpc-health-probe --help
# 对指定服务监控检查，服务名必须正确 完整：服务包名.服务名
grpc-health-probe -addr=127.0.0.1:58688 -service="kelvins_template.YourService"
# 对整体服务健康检查
grpc-health-probe -addr=127.0.0.1:58688 -service=""
```
3. 基于http方式请求RPC服务（前提是注册了rpc-gateway），http服务   
```shell
# 获取rpc-gateway header
# 根据proto定义的扩展选项拼接URL
#        option (google.api.http) = {
#            get: "/v1/service3"
#        };
curl http://service_name:service_port/v1/service3?id=100 -i -H 'authorization:Bearer v1.5da4f611ce5bc2052b303f6310e98b60941641498e33095c46e7aba17cadbfa9.1632475768'
HTTP/1.1 200 OK
Content-Type: application/json
Grpc-Metadata-Content-Type: application/grpc
Grpc-Metadata-Trailer: Grpc-Status
Grpc-Metadata-Trailer: Grpc-Message
Grpc-Metadata-Trailer: Grpc-Status-Details-Bin
Grpc-Metadata-X-Service-Name: kelvins-template
Grpc-Metadata-X-Service-Node: 192.168.0.101(hostname)
Grpc-Metadata-X-Powered-By: kelvins/rpc 1.5.10
Grpc-Metadata-X-Request-Id: 5664beaa-1c43-4eec-98d7-c611972c7303
Trailer: Grpc-Trailer-X-Response-Time
Trailer: Grpc-Trailer-X-Handle-Time
Date: Mon, 20 Sep 2021 08:11:19 GMT
Transfer-Encoding: chunked
{"common":{"msg":"99"},"result":"service3 "}
Grpc-Trailer-X-Handle-Time: 0.002057175/s
Grpc-Trailer-X-Response-Time: 2021-09-20 16:11:19.776

# 获取http服务的header
curl http://service_name:service_port/index -i
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8
X-Handle-Time: 0.000097/s
X-Service-Name: kelvins-template-http
X-Powered-By: kelvins/http(gin) 1.5.10
X-Request-Id: 69b8e804-4c5b-4091-b485-88400fceedc7
X-Response-Time: 2021-09-20 16:19:39.783
Date: Mon, 20 Sep 2021 08:19:39 GMT
Content-Length: 105
```

4. 在线服务（rpc，http）调用接口，使用wireshark抓包   
rpc服务调用 /kelvins_template.YourService/Service3 =>   
![avatar](./kelvins-rpc调用抓包.png)
http/h2服务调用 /hello接口 =>   
进行http2调用需要客户端发起协议升级，配置http2 transport   
![avatar](./kelvins-http开启H2调用抓包.png)

### 更新日志
时间 | 内容 |  贡献者 | 备注  
---|------|------|---
2020-6-1 | 基础开发阶段 | https://gitee.com/cristiane | 结构思考，基础搭建
2020-8-27 | 预览版上线 | https://gitee.com/cristiane | 支持gRPC，HTTP，crontab，queue类应用
2020-9-10 | 增加MongoDB支持 | https://gitee.com/cristiane | 基于配置加载MongoDB来初始化应用
2020-9-13 | 支持Redis队列 | https://gitee.com/cristiane | 基于配置加载queue-Redis来初始化应用
2020-11-24 | 若干更新 | https://gitee.com/cristiane | 若干更新
2021-4-5 | 支持应用优雅重启，退出 | https://gitee.com/cristiane | 基于操作系统信号，各平台有差异
2021-4-19 | 支持gin | https://gitee.com/cristiane | 允许将gin http handler注册到应用
2021-7-9 | 兼容Windows | https://gitee.com/cristiane | 修复Windows平台应用不能启动问题
2021-8-1 | 应用退出执行函数优化 | https://gitee.com/cristiane | 应用退出时异常处理
2021-8-1 | 应用支持负载均衡 | https://gitee.com/cristiane | 针对gRPC，http应用；同一应用多实例自动负载均衡
2021-8-7 | 启动命令 | https://gitee.com/cristiane | -s启动参数，支持启动进程，重启进程，停止进程
2021-8-13 | RPC健康检查 | https://gitee.com/cristiane | 支持使用grpc-health-probe等工具进行健康检查
2021-8-14 | RPC接入授权-token | https://gitee.com/cristiane | RPC应用支持开启接入授权
2021-8-14 | RPC-ghz压测试工具 | https://gitee.com/cristiane | 支持对RPC应用进行压力测试并输出报告
2021-9-1 | 若干更新 | https://gitee.com/cristiane | rpc日志对齐&rpc server参数配置化&启动优化
2021-9-11 | 若干更新 | https://gitee.com/cristiane | client_service,print,queue等若干优化
2021-9-18 | 运行环境优化，http优化 | https://gitee.com/cristiane | 根据运行环境日志打印
2021-9-20 | rpc，http注入metadata，服务注册优化 | https://gitee.com/cristiane | 诸如request-id，version，请求耗时，服务名，服务节点等
2021-9-25 | 重构rpc服务拦截器，http服务支持启用H2 | https://gitee.com/cristiane | 拦截器，http2
2021-10-1 | RPC，http服务支持限流器 | https://gitee.com/cristiane | 限流
2021-10-15 | watch在线服务在线状态 | https://gitee.com/cristiane | 监听etcd服务节点触发resolver
2021-10-15 | 在线服务注册etcd增加所处网络IP，服务类型 | https://gitee.com/cristiane | 不需要再单独配置服务host



### 业务应用
micro-mall-api系列共计22+个服务：https://gitee.com/cristiane/micro-mall-api

###技术交流
QQ群：852053097   
![avatar](./交流群.JPG)   
邮件：1225807604@qq.com