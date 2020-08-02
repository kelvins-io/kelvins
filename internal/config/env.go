package config

import (
	"os"
)

const (
	// ETCD V3 Server URL
	ENV_ETCDV3_SERVER_URL = "ETCDV3_SERVER_URL"
	// ETCD V3 Server URLs
	ENV_ETCDV3_SERVER_URLS = "ETCDV3_SERVER_URLS"
	// GO_ENV
	GO_ENV = "GO_ENV"
)

// GetEtcdV3ServerURL gets etcd v3 server url config from env.
func GetEtcdV3ServerURL() string {
	return os.Getenv(ENV_ETCDV3_SERVER_URL)
}

// GetEtcdV3ServerURLs gets etcd v3 server urls config from env.
func GetEtcdV3ServerURLs() string {
	values := os.Getenv(ENV_ETCDV3_SERVER_URLS)
	if values != "" {
		return values
	}

	return GetEtcdV3ServerURL()
}
