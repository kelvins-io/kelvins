package middleware

import (
	"container/list"
	"context"
	"gitee.com/kelvins-io/common/json"
	"gitee.com/kelvins-io/common/log"
	"gitee.com/kelvins-io/kelvins"
	"gitee.com/kelvins-io/kelvins/util/rpc_helper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"sync"
	"sync/atomic"
	"time"
)

type RPCRateLimitInterceptor struct {
	limiter *kelvinsRateLimit
}

func NewRPCRateLimitInterceptor(maxConcurrent, maxWaitNum, maxWaitSecond int, logger log.LoggerContextIface) *RPCRateLimitInterceptor {
	limiter := &kelvinsRateLimit{
		logger: logger,
	}
	if maxWaitNum > 0 {
		limiter.maxWaitNum = maxWaitNum
		limiter.maxWaitSecond = maxWaitSecond
		limiter.waitQueue = list.New()
		go limiter.loopQueue()
	}
	if maxConcurrent > 0 {
		limiter.maxConcurrent = maxConcurrent
		limiter.tickets = make(chan struct{}, maxConcurrent)
	}
	return &RPCRateLimitInterceptor{
		limiter: limiter,
	}
}

func (r *RPCRateLimitInterceptor) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if r.limiter.Limit() {
			requestMeta := rpc_helper.GetRequestMetadata(stream.Context())
			return status.Errorf(codes.ResourceExhausted, "%s requestMeta:%v is rejected by grpc_ratelimit middleware, please retry later.", info.FullMethod, json.MarshalToStringNoError(requestMeta))
		}
		defer func() {
			r.limiter.returnTicket()
		}()
		return handler(srv, stream)
	}
}

func (r *RPCRateLimitInterceptor) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if r.limiter.Limit() {
			requestMeta := rpc_helper.GetRequestMetadata(ctx)
			return nil, status.Errorf(codes.ResourceExhausted, "%s requestMeta:%v is rejected by grpc_ratelimit middleware, please retry later.", info.FullMethod, json.MarshalToStringNoError(requestMeta))
		}
		defer func() {
			r.limiter.returnTicket()
		}()
		return handler(ctx, req)
	}
}

type kelvinsRateLimit struct {
	maxConcurrent int
	maxWaitNum    int
	maxWaitSecond int
	tickets       chan struct{}
	ticketsState  int32
	waitQueue     *list.List
	waitQueueLock sync.RWMutex
	started       int32
	logger        log.LoggerContextIface
}

func (r *kelvinsRateLimit) Limit() bool {
	// no limit
	if r.maxConcurrent == 0 {
		return false
	}
	// take ticket
	take := r.takeTicket()
	if take {
		return false
	}
	if r.maxWaitNum == 0 {
		return false
	}
	notify := make(chan struct{}, 1)
	maxWaitSecond := 5 * time.Second
	if r.maxWaitSecond > 0 {
		maxWaitSecond = time.Duration(r.maxWaitSecond) * time.Second
	}
	atomic.StoreInt32(&r.started, 1)
	en := r.enQueue(&rateWaiteMeta{lastTime: time.Now().Add(maxWaitSecond), notify: notify})
	if !en {
		return true
	}
	_, ok := <-notify
	if ok {
		return false
	}

	return true
}

func (r *kelvinsRateLimit) loopQueue() {
	if r.maxWaitNum == 0 {
		return
	}
	if r.waitQueue == nil {
		return
	}
	var isFirst = true
	var curEle *list.Element
	fn := func() {
		if atomic.LoadInt32(&r.started) == 0 {
			return
		}
		if isFirst {
			r.waitQueueLock.RLock()
			curEle = r.waitQueue.Front()
			r.waitQueueLock.RUnlock()
			isFirst = false
		} else {
			if curEle != nil {
				curEle = curEle.Next()
			}
		}
		if curEle == nil {
			return
		}
		meta := curEle.Value.(*rateWaiteMeta)
		if meta == nil {
			return
		}
		r.logger.Info(context.TODO(), "loopQueue 取出了一个有效节点")
		if meta.lastTime.Before(time.Now()) {
			r.waitQueueLock.Lock()
			r.waitQueue.Remove(curEle)
			r.waitQueueLock.Unlock()
			atomic.StoreInt32(&meta.notifyState, 1)
			close(meta.notify)
			return
		}
		r.logger.Info(context.TODO(), "loopQueue 没有超时")
		take := r.takeTicket()
		if take {
			r.logger.Info(context.TODO(), "loopQueue takeTicket")
			if atomic.LoadInt32(&meta.notifyState) == 0 {
				meta.notify <- struct{}{}
			}
			r.waitQueueLock.Lock()
			r.waitQueue.Remove(curEle)
			r.waitQueueLock.Unlock()
		}
	}

	ticker := time.NewTicker(50 * time.Millisecond)
	for {
		select {
		case <-kelvins.AppCloseCh:
			atomic.StoreInt32(&r.ticketsState, 1)
			ticker.Stop()
			return
		case <-ticker.C:
			fn()
		}
	}
}

func (r *kelvinsRateLimit) enQueue(meta *rateWaiteMeta) bool {
	if r.maxWaitNum == 0 {
		return false
	}
	if r.waitQueue == nil {
		return false
	}
	select {
	case <-kelvins.AppCloseCh:
		return false
	default:
	}
	r.logger.Info(context.TODO(), "enQueue 准备获取锁")
	r.waitQueueLock.RLock()
	queueLen := r.waitQueue.Len()
	r.waitQueueLock.RUnlock()
	if queueLen >= r.maxWaitNum {
		return false
	}
	r.logger.Info(context.TODO(), "enQueue 获取锁 ok")
	r.waitQueueLock.Lock()
	r.waitQueue.PushBack(meta)
	r.waitQueueLock.Unlock()
	return true
}

func (r *kelvinsRateLimit) takeTicket() bool {
	if r.maxConcurrent == 0 {
		return true
	}
	if r.tickets == nil {
		return true
	} else {
		if atomic.LoadInt32(&r.ticketsState) == 1 {
			return false
		}
	}

	select {
	case <-kelvins.AppCloseCh:
		return false
	case r.tickets <- struct{}{}:
		r.logger.Info(context.TODO(), "takeTicket")
		return true
	default:
		return false
	}
}

func (r *kelvinsRateLimit) returnTicket() {
	if r.maxConcurrent == 0 {
		return
	}
	if r.maxWaitNum == 0 {
		return
	}
	if r.tickets == nil {
		return
	}
	r.logger.Info(context.TODO(), "returnTicket")
	select {
	case <-r.tickets:
	default:
	}
}

type rateWaiteMeta struct {
	lastTime    time.Time
	notify      chan struct{}
	notifyState int32
}
