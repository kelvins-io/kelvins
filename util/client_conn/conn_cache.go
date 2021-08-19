package client_conn

import (
	"errors"
	"fmt"
	"github.com/bluele/gcache"
	"google.golang.org/grpc"
	"math"
	"time"
)

var (
	internalCache    gcache.Cache
	internalNotFound = errors.New("object not found")
)

const (
	internalCachePrefix = "kelvins-%v"
	internalCacheExpire = 24 * time.Hour
)

func init() {
	internalCache = gcache.New(math.MaxInt16).LRU().Build()
}

func getRPCConn(serviceName string) (conn *grpc.ClientConn, err error) {
	key := genInternalCacheKey(serviceName)
	exist := internalCache.Has(key)
	if exist {
		value, err := internalCache.Get(key)
		if err != nil {
			return
		}
		connCache, ok := value.(*grpc.ClientConn)
		if ok {
			conn = connCache
			return
		}
	}
	err = internalNotFound
	return
}

func storageRPCConn(serviceName string, conn *grpc.ClientConn) error {
	key := genInternalCacheKey(serviceName)
	return internalCache.Set(key, conn)
}

func genInternalCacheKey(serviceName string) string {
	return fmt.Sprintf(internalCachePrefix, serviceName)
}
