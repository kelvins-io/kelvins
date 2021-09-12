package setup

import (
	"fmt"
	"gitee.com/kelvins-io/kelvins/config/setting"
	"time"

	"github.com/gomodule/redigo/redis"
)

// NewRedis returns *redis.Pool instance.
func NewRedis(redisSetting *setting.RedisSettingS) (*redis.Pool, error) {
	if redisSetting == nil {
		return nil, fmt.Errorf("RedisSetting is nil")
	}
	if redisSetting.Host == "" {
		return nil, fmt.Errorf("Lack of redisSetting.Host")
	}
	if redisSetting.Password == "" {
		return nil, fmt.Errorf("Lack of redisSetting.Password")
	}

	maxIdle := redisSetting.MaxIdle
	maxActive := redisSetting.MaxActive
	if redisSetting.MaxActive > 0 && redisSetting.MaxIdle > 0 {
		maxIdle = redisSetting.MaxIdle
		maxActive = redisSetting.MaxActive
	}
	idleTimeout := 240
	if redisSetting.IdleTimeout > 0 {
		idleTimeout = redisSetting.IdleTimeout
	}
	return &redis.Pool{
		MaxIdle:     maxIdle,
		MaxActive:   maxActive,
		IdleTimeout: time.Duration(idleTimeout) * time.Second,
		Dial: func() (redis.Conn, error) {
			var opts []redis.DialOption
			if redisSetting.ConnectTimeout > 0 {
				opts = append(opts, redis.DialConnectTimeout(time.Duration(redisSetting.ConnectTimeout)*time.Second))
			}
			if redisSetting.ReadTimeout > 0 {
				opts = append(opts, redis.DialReadTimeout(time.Duration(redisSetting.ReadTimeout)*time.Second))
			}
			if redisSetting.WriteTimeout > 0 {
				opts = append(opts, redis.DialWriteTimeout(time.Duration(redisSetting.WriteTimeout)*time.Second))
			}
			c, err := redis.Dial("tcp", redisSetting.Host, opts...)
			if err != nil {
				return nil, err
			}
			if redisSetting.Password != "" {
				if _, err := c.Do("AUTH", redisSetting.Password); err != nil {
					c.Close()
					return nil, err
				}
			}
			if redisSetting.DB > 0 {
				if _, err := c.Do("SELECT", redisSetting.DB); err != nil {
					c.Close()
					return nil, err
				}
			}

			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}, nil
}
