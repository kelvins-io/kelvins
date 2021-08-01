package etcdconfig

import (
	"context"
	"errors"
	"fmt"
	"gitee.com/kelvins-io/common/json"
	"gitee.com/kelvins-io/kelvins"
	"gitee.com/kelvins-io/kelvins/internal/service/slb"
	"gitee.com/kelvins-io/kelvins/internal/util"
	"github.com/coreos/etcd/client"
	"strings"
	"time"
)

const (
	ROOT            = "/"
	SERVICE         = "kelvins-service"
	DEFAULT_CLUSTER = "load-balance"
)

var ErrServiceConfigKeyNotExist = errors.New("service config key not exist")

type ServiceConfigClient struct {
	ServiceLB *slb.ServiceLB
	Config
}

type Config struct {
	ServiceVersion string `json:"service_version"`
	ServicePort    string `json:"service_port"`
}

func NewServiceConfigClient(slb *slb.ServiceLB) *ServiceConfigClient {
	return &ServiceConfigClient{ServiceLB: slb}
}

func (s *ServiceConfigClient) GetKeyName(serverName string, sequences ...string) string {
	key := ROOT + SERVICE + "." + serverName + "." + DEFAULT_CLUSTER + "@" + kelvins.Version
	for _, s := range sequences {
		key += "/" + s
	}
	return key
}

func (s *ServiceConfigClient) GetConfig(sequence string) (*Config, error) {
	cli, err := util.NewEtcd(s.ServiceLB.EtcdServerUrl)
	if err != nil {
		return nil, fmt.Errorf("util.NewEtcdKeysAPI err: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	key := s.GetKeyName(s.ServiceLB.ServerName, sequence)
	serviceInfo, err := client.NewKeysAPI(cli).Get(ctx, key, nil)
	if err != nil {
		if client.IsKeyNotFound(err) {
			return nil, ErrServiceConfigKeyNotExist
		}
		return nil, fmt.Errorf("cli.Get err: %v, key: %v", err, key)
	}

	var config Config
	if len(serviceInfo.Node.Value) > 0 {
		err = json.Unmarshal(serviceInfo.Node.Value, &config)
		if err != nil {
			return nil, fmt.Errorf("json.Unmarshal err: %v, key: %v,values: %v", err, key, serviceInfo.Node.Value)
		}
	}

	if config.ServicePort == "" {
		return nil, fmt.Errorf("servicePort is empty, key: %s", key)
	}

	return &config, nil
}

func (s *ServiceConfigClient) ClearConfig(sequence string) error {
	cli, err := util.NewEtcd(s.ServiceLB.EtcdServerUrl)
	if err != nil {
		return fmt.Errorf("util.NewEtcdKeysAPI err: %v", err)
	}

	key := s.GetKeyName(s.ServiceLB.ServerName, sequence)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = client.NewKeysAPI(cli).Delete(ctx, key, nil)
	if err != nil {
		if client.IsKeyNotFound(err) {
			return ErrServiceConfigKeyNotExist
		}
		return fmt.Errorf("cli.Delete err: %v key: %v", err, key)
	}

	return nil
}

func (s *ServiceConfigClient) WriteConfig(sequence string, c Config) error {
	cli, err := util.NewEtcd(s.ServiceLB.EtcdServerUrl)
	if err != nil {
		return fmt.Errorf("util.NewEtcdKeysAPI err: %v", err)
	}

	key := s.GetKeyName(s.ServiceLB.ServerName, sequence)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	jsonConfig, err := json.MarshalToString(&c)
	if err != nil {
		return fmt.Errorf("json.MarshalToString err: %v key: %v config: %+v", err, key, c)
	}
	_, err = client.NewKeysAPI(cli).Set(ctx, key, jsonConfig, &client.SetOptions{
		PrevExist: client.PrevNoExist,
	})
	if err != nil {
		return fmt.Errorf("cli.Set err: %v key: %v values: %v", err, key, jsonConfig)
	}

	return nil
}

func (s *ServiceConfigClient) ListConfigs() (map[string]*Config, error) {
	return s.listConfigs("/")
}

func (s *ServiceConfigClient) GetConfigs() (map[string]*Config, error) {
	return s.listConfigs(s.GetKeyName(s.ServiceLB.ServerName))
}

func (s *ServiceConfigClient) listConfigs(key string) (map[string]*Config, error) {
	cli, err := util.NewEtcd(s.ServiceLB.EtcdServerUrl)
	if err != nil {
		return nil, fmt.Errorf("util.NewEtcdKeysAPI err: %v", err)
	}

	kapi := client.NewKeysAPI(cli)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	serviceInfos, err := kapi.Get(ctx, key, nil)
	if err != nil {
		if client.IsKeyNotFound(err) {
			return nil, ErrServiceConfigKeyNotExist
		}
		return nil, fmt.Errorf("cli.Get err: %v key: %v", err, key)
	}

	configs := make(map[string]*Config)
	for _, info := range serviceInfos.Node.Nodes {
		if len(info.Value) > 0 {
			index := strings.Index(info.Key, ROOT+SERVICE)
			if index == 0 {
				config := &Config{}
				err := json.Unmarshal(info.Value, config)
				if err != nil {
					return nil, fmt.Errorf("json.UnmarshalByte err: %v values: %v", err, info.Value)
				}

				configs[string(info.Key)] = config
			}
		}
	}

	return configs, nil
}
