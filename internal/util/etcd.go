package util

import (
	"strings"
	"time"

	"github.com/etcd-io/etcd/client"
)

func NewEtcd(urls string) (client.Client, error) {
	cli, err := client.New(client.Config{
		Endpoints:               strings.Split(urls, ","),
		Transport:               client.DefaultTransport,
		HeaderTimeoutPerRequest: 10 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	return cli, nil
}
