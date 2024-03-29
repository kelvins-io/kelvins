package etcdconfig

import (
	"context"
	"errors"
	"fmt"
	"gitee.com/kelvins-io/common/json"
	"gitee.com/kelvins-io/kelvins/internal/service/slb"
	"gitee.com/kelvins-io/kelvins/internal/util"
	"gitee.com/kelvins-io/kelvins/internal/vars"
	"github.com/etcd-io/etcd/client"
	"strings"
	"time"
)

const (
	Service        = "/kelvins-service"
	DefaultCluster = "cluster"
)

var ErrServiceConfigKeyNotExist = errors.New("service config key not exist")

type ServiceConfigClient struct {
	ServiceLB *slb.ServiceLB
	Config
}

type Config struct {
	ServiceVersion string `json:"service_version"`
	ServicePort    string `json:"service_port"`
	ServiceIP      string `json:"service_ip"`
	ServiceKind    string `json:"service_kind"`
	LastModified   string `json:"last_modified"`
}

func NewServiceConfigClient(slb *slb.ServiceLB) *ServiceConfigClient {
	return &ServiceConfigClient{ServiceLB: slb}
}

// GetKeyName etcd key cannot end with a number
func (s *ServiceConfigClient) GetKeyName(serverName string, sequences ...string) string {
	key := Service + "." + serverName + "." + DefaultCluster
	for _, s := range sequences {
		key += "/" + s
	}
	return key
}

func (s *ServiceConfigClient) GetConfig(sequence string) (*Config, error) {
	cli, err := util.NewEtcd(s.ServiceLB.EtcdServerUrl)
	if err != nil {
		return nil, fmt.Errorf("util.NewEtcd err: %v，etcdUrl: %v", err, s.ServiceLB.EtcdServerUrl)
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
		return fmt.Errorf("util.NewEtcd err: %v，etcdUrl: %v", err, s.ServiceLB.EtcdServerUrl)
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
		return fmt.Errorf("util.NewEtcd err: %v，etcdUrl: %v", err, s.ServiceLB.EtcdServerUrl)
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

func (s *ServiceConfigClient) Watch(ctx context.Context) (<-chan struct{}, error) {
	notice := make(chan struct{}, 1)
	cli, err := util.NewEtcd(s.ServiceLB.EtcdServerUrl)
	if err != nil {
		return notice, fmt.Errorf("util.NewEtcd err: %v，etcdUrl: %v", err, s.ServiceLB.EtcdServerUrl)
	}

	kapi := client.NewKeysAPI(cli)
	ctx, cancel := context.WithCancel(ctx)
	watcher := kapi.Watcher(s.GetKeyName(s.ServiceLB.ServerName), &client.WatcherOptions{
		AfterIndex: 0,
		Recursive:  true,
	})
	go func() {
		defer func() {
			cancel()
			close(notice)
		}()
		for {
			select {
			case <-ctx.Done():
				return
			case <-vars.AppCloseCh:
				return
			default:
			}
			resp, err := watcher.Next(ctx)
			if err != nil {
				time.Sleep(500 * time.Millisecond)
				continue
			}

			if strings.ToLower(resp.Action) != "get" {
				// 防止notice来不及被客户端消费
				select {
				case <-notice:
				default:
				}
				notice <- struct{}{}
			}
		}
	}()

	return notice, nil
}

func (s *ServiceConfigClient) listConfigs(key string) (map[string]*Config, error) {
	cli, err := util.NewEtcd(s.ServiceLB.EtcdServerUrl)
	if err != nil {
		return nil, fmt.Errorf("util.NewEtcd err: %v，etcdUrl: %v", err, s.ServiceLB.EtcdServerUrl)
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
			index := strings.Index(info.Key, Service)
			if index == 0 {
				config := &Config{}
				err := json.Unmarshal(info.Value, config)
				if err != nil {
					return nil, fmt.Errorf("json.UnmarshalByte err: %v values: %v", err, info.Value)
				}

				configs[info.Key] = config
			}
		}
	}

	return configs, nil
}
