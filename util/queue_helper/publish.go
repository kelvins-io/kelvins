package queue_helper

import (
	"context"
	"fmt"
	"gitee.com/kelvins-io/common/errcode"
	"gitee.com/kelvins-io/common/json"
	"gitee.com/kelvins-io/common/log"
	"gitee.com/kelvins-io/common/queue"
	"gitee.com/kelvins-io/kelvins/internal/logging"
	"github.com/RichardKnop/machinery/v1/tasks"
)

type PublishService struct {
	logger log.LoggerContextIface
	server *queue.MachineryQueue
	tag    *PushMsgTag
}

type PushMsgTag struct {
	DeliveryTag    string // consume func name
	DeliveryErrTag string // consume err func name
	RetryCount     int    // default 3 cycle
	RetryTimeout   int    // default 10s
}

func NewPublishService(server *queue.MachineryQueue, tag *PushMsgTag, logger log.LoggerContextIface) (*PublishService, error) {
	if server == nil {
		return nil, fmt.Errorf("server nil !!! ")
	}
	if tag == nil {
		return nil, fmt.Errorf("tag nil !!! ")
	}
	if tag.DeliveryTag == "" {
		return nil, fmt.Errorf("tag.DeliveryTag empty !!! ")
	}
	if tag.DeliveryErrTag == "" {
		return nil, fmt.Errorf("tag.DeliveryErrTag empty !!! ")
	}
	if tag.RetryCount <= 0 {
		tag.RetryCount = 3
	}
	if tag.RetryTimeout <= 0 {
		tag.RetryTimeout = 10
	}
	return &PublishService{
		server: server,
		tag:    tag,
		logger: logger,
	}, nil
}

func (p *PublishService) PushMessage(ctx context.Context, args interface{}) (string, int) {

	taskSign, retCode := p.buildQueueData(ctx, args)
	if retCode != errcode.SUCCESS {
		return "", retCode
	}

	taskId, retCode := p.sendTaskToQueue(ctx, taskSign)
	if retCode != errcode.SUCCESS {
		return "", retCode
	}

	return taskId, errcode.SUCCESS
}

// 构建队列数据
func (p *PublishService) buildQueueData(ctx context.Context, args interface{}) (*tasks.Signature, int) {

	sign := p.buildTaskSignature(args)

	errSign, err := tasks.NewSignature(
		p.tag.DeliveryErrTag, []tasks.Arg{
			{
				Name:  "data",
				Type:  "string",
				Value: json.MarshalToStringNoError(args),
			},
		})

	if err != nil {
		if p.logger != nil {
			p.logger.Errorf(ctx, "queue_helper buildQueueData err: %v, taskSign: %v", err, json.MarshalToStringNoError(sign))
		} else {
			logging.Errf("queue_helper buildQueueData err: %v, taskSign: %v\n", err, json.MarshalToStringNoError(sign))
		}
		return nil, errcode.FAIL
	}

	errCallback := make([]*tasks.Signature, 0)
	errCallback = append(errCallback, errSign)
	sign.OnError = errCallback

	return sign, errcode.SUCCESS
}

// 构建任务签名
func (p *PublishService) buildTaskSignature(args interface{}) *tasks.Signature {

	taskSignature := &tasks.Signature{
		Name:         p.tag.DeliveryTag,
		RetryCount:   p.tag.RetryCount,
		RetryTimeout: p.tag.RetryTimeout,
		Args: []tasks.Arg{
			{
				Name:  "data",
				Type:  "string",
				Value: json.MarshalToStringNoError(args),
			},
		},
	}

	return taskSignature
}

// 将任务发送到队列
func (p *PublishService) sendTaskToQueue(ctx context.Context, taskSign *tasks.Signature) (string, int) {

	result, err := p.server.TaskServer.SendTaskWithContext(ctx, taskSign)
	if err != nil {
		if p.logger != nil {
			p.logger.Errorf(ctx, "queue_helper sendTaskToQueue err:%v, data:%v", err, json.MarshalToStringNoError(taskSign))
		} else {
			logging.Errf("queue_helper sendTaskToQueue err:%v, data:%v\n", err, json.MarshalToStringNoError(taskSign))
		}
		return "", errcode.FAIL
	}

	return result.Signature.UUID, errcode.SUCCESS
}

func (p *PublishService) GetTaskState(taskId string) (*tasks.TaskState, error) {

	return p.server.TaskServer.GetBackend().GetState(taskId)
}
