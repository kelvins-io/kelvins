package setup

import (
	"fmt"
	"gitee.com/kelvins-io/common/queue"
	"gitee.com/kelvins-io/kelvins/config/setting"
)

// NewRedisQueue returns *kelvinsqueue.MachineryQueue instance of redis queue.
func NewRedisQueue(redisQueueSetting *setting.QueueRedisSettingS, namedTaskFuncs map[string]interface{}) (*queue.MachineryQueue, error) {
	if redisQueueSetting == nil {
		return nil, fmt.Errorf("RedisQueueSetting is nil")
	}
	if redisQueueSetting.Broker == "" {
		return nil, fmt.Errorf("Lack of redisQueueSetting.Broker")
	}
	if redisQueueSetting.DefaultQueue == "" {
		return nil, fmt.Errorf("Lack of redisQueueSetting.DefaultQueue")
	}
	if redisQueueSetting.ResultBackend == "" {
		return nil, fmt.Errorf("Lack of redisQueueSetting.ResultBackend")
	}
	if redisQueueSetting.ResultsExpireIn < 0 {
		return nil, fmt.Errorf("RedisQueueSetting.ResultsExpireIn must >= 0")
	}

	redisQueue, err := queue.NewRedisQueue(
		redisQueueSetting.Broker,
		redisQueueSetting.DefaultQueue,
		redisQueueSetting.ResultBackend,
		redisQueueSetting.ResultsExpireIn,
		namedTaskFuncs,
	)
	if err != nil {
		return nil, fmt.Errorf("kelvinsqueue.NewRedisQueue: %v", err)
	}

	return redisQueue, nil
}

// NewAliAMQPQueue returns *kelvinsqueue.MachineryQueue instance of aliyun AMQP queue.
func NewAliAMQPQueue(aliAMQPQueueSetting *setting.QueueAliAMQPSettingS, namedTaskFuncs map[string]interface{}) (*queue.MachineryQueue, error) {
	if aliAMQPQueueSetting == nil {
		return nil, fmt.Errorf("AliAMQPQueueSetting is nil")
	}
	if aliAMQPQueueSetting.AccessKey == "" {
		return nil, fmt.Errorf("Lack of aliAMQPQueueSetting.AccessKey")
	}
	if aliAMQPQueueSetting.SecretKey == "" {
		return nil, fmt.Errorf("Lack of aliAMQPQueueSetting.SecretKey")
	}
	if aliAMQPQueueSetting.AliUid < 0 {
		return nil, fmt.Errorf("AliAMQPQueueSetting.AliUid must >= 0")
	}
	if aliAMQPQueueSetting.EndPoint == "" {
		return nil, fmt.Errorf("Lack of aliAMQPQueueSetting.EndPoint")
	}
	if aliAMQPQueueSetting.VHost == "" {
		return nil, fmt.Errorf("Lack of aliAMQPQueueSetting.VHost")
	}
	if aliAMQPQueueSetting.DefaultQueue == "" {
		return nil, fmt.Errorf("Lack of aliAMQPQueueSetting.DefaultQueue")
	}
	if aliAMQPQueueSetting.ResultBackend == "" {
		return nil, fmt.Errorf("Lack of aliAMQPQueueSetting.ResultBackend")
	}
	if aliAMQPQueueSetting.ResultsExpireIn < 0 {
		return nil, fmt.Errorf("AliAMQPQueueSetting.ResultsExpireIn must >= 0")
	}

	var aliAMQPConfig = &queue.AliAMQPConfig{
		AccessKey:        aliAMQPQueueSetting.AccessKey,
		SecretKey:        aliAMQPQueueSetting.SecretKey,
		AliUid:           aliAMQPQueueSetting.AliUid,
		EndPoint:         aliAMQPQueueSetting.EndPoint,
		VHost:            aliAMQPQueueSetting.VHost,
		DefaultQueue:     aliAMQPQueueSetting.DefaultQueue,
		ResultBackend:    aliAMQPQueueSetting.ResultBackend,
		ResultsExpireIn:  aliAMQPQueueSetting.ResultsExpireIn,
		Exchange:         aliAMQPQueueSetting.Exchange,
		ExchangeType:     aliAMQPQueueSetting.ExchangeType,
		BindingKey:       aliAMQPQueueSetting.BindingKey,
		PrefetchCount:    aliAMQPQueueSetting.PrefetchCount,
		QueueBindingArgs: nil,
		NamedTaskFuncs:   namedTaskFuncs,
	}

	var aliAMQPQueue, err = queue.NewAliAMQPMqQueue(aliAMQPConfig)
	if err != nil {
		return nil, fmt.Errorf("kelvinsqueue.NewAliAMQPMqQueue: %v", err)
	}

	return aliAMQPQueue, nil
}

// SetUpAMQPQueue returns *queue.MachineryQueue instance of  AMQP queue.
func NewAMQPQueue(amqpQueueSetting *setting.QueueAMQPSettingS, namedTaskFuncs map[string]interface{}) (*queue.MachineryQueue, error) {
	if amqpQueueSetting == nil {
		return nil, fmt.Errorf("[err] amqpQueueSetting is nil")
	}
	if amqpQueueSetting.Broker == "" {
		return nil, fmt.Errorf("[err] Lack of amqpQueueSetting.Broker")
	}
	if amqpQueueSetting.DefaultQueue == "" {
		return nil, fmt.Errorf("[err] Lack of amqpQueueSetting.DefaultQueue")
	}
	if amqpQueueSetting.ResultBackend == "" {
		return nil, fmt.Errorf("[err] Lack of amqpQueueSetting.ResultBackend")
	}
	if amqpQueueSetting.ResultsExpireIn < 0 {
		return nil, fmt.Errorf("[err] amqpQueueSetting.ResultsExpireIn must >= 0")
	}

	var amqpQueue, err = queue.NewRabbitMqQueue(
		amqpQueueSetting.Broker,
		amqpQueueSetting.DefaultQueue,
		amqpQueueSetting.ResultBackend,
		amqpQueueSetting.ResultsExpireIn,
		amqpQueueSetting.Exchange,
		amqpQueueSetting.ExchangeType,
		amqpQueueSetting.BindingKey,
		amqpQueueSetting.PrefetchCount,
		nil,
		namedTaskFuncs)
	if err != nil {
		return nil, fmt.Errorf("kelvinsqueue.NewAliAMQPMqQueue: %v", err)
	}

	return amqpQueue, nil
}
