package setup

import (
	"fmt"
	"gitee.com/kelvins-io/kelvins/config/setting"
	"google.golang.org/grpc"
)

// NewGRPC ...
func NewGRPC(serverSetting *setting.ServerSettingS, opts []grpc.ServerOption) (*grpc.Server, error) {
	if serverSetting == nil {
		return nil, fmt.Errorf("serverSetting is nil")
	}

	return grpc.NewServer(opts...), nil
}
