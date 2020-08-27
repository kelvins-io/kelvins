package etcdconfig

import (
	"context"
	"fmt"
	"gitee.com/kelvins-io/common/json"
	"gitee.com/kelvins-io/kelvins/internal/service/slb"
	"gitee.com/kelvins-io/kelvins/internal/util"
	"github.com/coreos/etcd/client"
	"strings"
	"time"
)

const (
	ROOT            = "/"
	SERVICE         = "service"
	DEFALUT_CLUSTER = "default"
)

type ServiceConfig struct {
	ServiceLB *slb.ServiceLB
	Config
}

type Config struct {
	ServiceVersion string `json:"service_version"`
	ServicePort    string `json:"service_port"`
}

func NewServiceConfig(slb *slb.ServiceLB) *ServiceConfig {
	return &ServiceConfig{ServiceLB: slb}
}

func (s *ServiceConfig) GetKeyName(serverName string) string {
	return ROOT + SERVICE + "." + serverName + "." + DEFALUT_CLUSTER
}

func (s *ServiceConfig) GetConfig() (*Config, error) {
	cli, err := util.NewEtcd(s.ServiceLB.EtcdServerUrl)
	if err != nil {
		return nil, fmt.Errorf("util.NewEtcdKeysAPI err: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	key := s.GetKeyName(s.ServiceLB.ServerName)
	serviceInfo, err := client.NewKeysAPI(cli).Get(ctx, key, nil)
	if err != nil {
		return nil, fmt.Errorf("cli.Get err: %v", err)
	}

	var config Config
	if len(serviceInfo.Node.Value) > 0 {
		err = json.Unmarshal(serviceInfo.Node.Value, &config)
		if err != nil {
			return nil, fmt.Errorf("json.Unmarshal err: %v", err)
		}
	}

	if config.ServicePort == "" {
		return nil, fmt.Errorf("servicePort is empty, key: %s", key)
	}

	return &config, nil
}

func (s *ServiceConfig) WriteConfig(c Config) error {
	cli, err := util.NewEtcd(s.ServiceLB.EtcdServerUrl)
	if err != nil {
		return fmt.Errorf("util.NewEtcdKeysAPI err: %v", err)
	}

	key := s.GetKeyName(s.ServiceLB.ServerName)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sconfig, err := json.MarshalToString(&c)
	if err != nil {
		return fmt.Errorf("json.MarshalToString err: %v", err)
	}

	_, err = client.NewKeysAPI(cli).Set(ctx, key, sconfig, nil)
	if err != nil {
		return fmt.Errorf("cli.Put err: %v", err)
	}

	return nil
}

func (s *ServiceConfig) GetConfigs() (map[string]*Config, error) {
	cli, err := util.NewEtcd(s.ServiceLB.EtcdServerUrl)
	if err != nil {
		return nil, fmt.Errorf("util.NewEtcdKeysAPI err: %v", err)
	}

	kapi := client.NewKeysAPI(cli)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	serviceInfos, err := kapi.Get(ctx, "/", nil)
	if err != nil {
		return nil, fmt.Errorf("cli.Get err: %v", err)
	}

	configs := make(map[string]*Config)
	for _, info := range serviceInfos.Node.Nodes {
		if len(info.Value) > 0 {
			index := strings.Index(info.Key, ROOT+SERVICE)
			if index == 0 {
				config := &Config{}
				err := json.Unmarshal(info.Value, config)
				if err != nil {
					return nil, fmt.Errorf("json.UnmarshalByte err: %v", err)
				}

				configs[string(info.Key)] = config
			}
		}
	}

	return configs, nil
}
