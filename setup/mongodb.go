package setup

import (
	"context"
	"fmt"
	"gitee.com/kelvins-io/kelvins/config/setting"
	"github.com/qiniu/qmgo"
	"log"
)

func NewMongoDBClient(mongodbSetting *setting.MongoDBSettingS) (*qmgo.QmgoClient, error) {
	if mongodbSetting == nil {
		return nil, fmt.Errorf("mongodbSetting is nil")
	}
	if mongodbSetting.Uri == "" {
		return nil, fmt.Errorf("mongodbSetting.Uri is empty")
	}
	if mongodbSetting.Database == "" {
		return nil, fmt.Errorf("mongodbSetting.Database is empty")
	}
	if mongodbSetting.Username == "" {
		return nil, fmt.Errorf("mongodbSetting.Username is empty")
	}
	if mongodbSetting.Password == "" {
		return nil, fmt.Errorf("mongodbSetting.Password is empty")
	}
	if mongodbSetting.AuthSource == "" {
		return nil, fmt.Errorf("mongodbSetting.AuthSource is empty")
	}
	if mongodbSetting.MaxPoolSize <= 0 {
		return nil, fmt.Errorf("mongodbSetting.MaxPoolSize is letter or equal zero")
	}
	if mongodbSetting.MinPoolSize <= 0 {
		return nil, fmt.Errorf("mongodbSetting.MinPoolSize is letter or equal zero")
	}
	// 初始化mongodb
	ctx := context.Background()
	var maxPoolSize = uint64(mongodbSetting.MaxPoolSize)
	var minPoolSize = uint64(mongodbSetting.MinPoolSize)

	mgoCfg := &qmgo.Config{
		Uri:         mongodbSetting.Uri,
		Database:    mongodbSetting.Database,
		MaxPoolSize: &maxPoolSize,
		MinPoolSize: &minPoolSize,
		Auth: &qmgo.Credential{
			AuthMechanism: "",
			AuthSource:    mongodbSetting.AuthSource,
			Username:      mongodbSetting.Username,
			Password:      mongodbSetting.Password,
			PasswordSet:   false,
		},
	}
	client, err := qmgo.Open(ctx, mgoCfg)
	if err != nil {
		log.Printf("mongodb open err: %v\n", err)
		return nil, err
	}

	return client, nil
}
