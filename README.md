# kelvins

微服务框架

### 运行环境变量
etcd集群地址   
ETCDV3_SERVER_URLS     
```
笔者自己环境的配置（仅做参考） 
export ETCDCTL_API=3
export ETCDV3_SERVER_URL=http://10.211.55.4:2379,http://10.211.55.7:2379
如果自己搭建etcd集群最少需要两个节点（一主一从）本地搭建参考：https://gitee.com/cristiane/micro-mall-api/blob/master/%E5%BE%AE%E5%95%86%E5%9F%8EETCD%E9%83%A8%E7%BD%B2.pdf
```

GO_ENV
```
运行环境标识，可选值有：dev，test，release，prod；分别对应开发环境，测试环境，预发布/灰度环境，prod正式环境，本地配置export GO_ENV=dev就好
```

### 目前最新版支持的配置文件
``` 
配置文件默认就在项目根目录下/etc/app.ini文件，大多数micro-mall-开头的项目已经包含了所需的配置文件，根据自己本地的配置修改即可
```
截止最新版kelvins，全部支持的项目配置（./etc/app.ini）内容   
除了必要的配置项外，其余配置都是你不配置kelvins就不会去加载初始&当然对应的功能也不能用   
服务配置：端口，传输协议等   

**必要配置项：   
kelvins-server    
日志：级别，路径等   
kelvins-logger   

--自选配置项：   
MySQL：连接配置信息   
kelvins-mysql   
Redis：连接配置信息   
kelvins-redis   
MongoDB：连接配置信息   
kelvins-mongodb   
队列功能需要的Redis配置   
kelvins-queue-redis   
队列功能-阿里云队列（在阿里云购买的amqp）   
kelvins-queue-ali-amqp   
队列功能-amqp协议（也就是自己搭建的rabbitmq）   
kelvins-queue-amqp   
队列功能，事件订阅功能（阿里云）   
kelvins-queue-ali-rocketmq   
队列消费者配置   
kelvins-queue-server   

++自定义配置项，根据项目本身而定    
micro-mall-api/etcd/app.ini#EmailConfig就属于自定义配置项    

###技术交流
QQ群：578859618   
邮件：1225807604@qq.com