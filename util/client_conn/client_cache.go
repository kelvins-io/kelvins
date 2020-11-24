package client_conn

import (
	"context"
	"gitee.com/kelvins-io/kelvins"
	"gitee.com/kelvins-io/kelvins/internal/service/slb/etcdconfig"
	"github.com/bluele/gcache"
	"math"
	"time"
)

var (
	clientConfigCache gcache.Cache // 客户端缓存
	ctx               = context.Background()
)

func init() {
	clientConfigCache = gcache.New(math.MaxInt8).LRU().Build()
}

func storeClientConfig(serviceName string, config *etcdconfig.Config) {
	err := clientConfigCache.SetWithExpire(serviceName, *config, 5*time.Minute)
	if err != nil {
		kelvins.FrameworkLogger.Errorf(ctx, "[kelvins] storeClientConfig err: %v, serviceName: %v,config: %+v", err, serviceName, config)
		return
	}
}

func loadClientConfig(serviceName string) *etcdconfig.Config {
	exist := clientConfigCache.Has(serviceName)
	if exist {
		obj, err := clientConfigCache.Get(serviceName)
		if err == nil && obj != nil {
			c, ok := obj.(etcdconfig.Config)
			if ok {
				return &c
			}
		}
		return nil
	}
	return nil
}
