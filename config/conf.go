package config

import (
	"gitee.com/kelvins-io/kelvins/internal/config"
)

// MapConfig loads config to struct v.
func MapConfig(section string, v interface{}) {
	config.MapConfig(section, v)
}
