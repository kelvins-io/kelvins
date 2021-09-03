package client_conn

import (
	"context"
	"fmt"
	"gitee.com/kelvins-io/kelvins"
	"gitee.com/kelvins-io/kelvins/internal/config"
	"gitee.com/kelvins-io/kelvins/internal/logging"
	"gitee.com/kelvins-io/kelvins/internal/service/slb"
	"gitee.com/kelvins-io/kelvins/internal/service/slb/etcdconfig"
	"gitee.com/kelvins-io/kelvins/internal/vars"
	"google.golang.org/grpc/resolver"
	"time"
)

const (
	kelvinsScheme   = "kelvins-scheme"
	minResolverRate = 5 * time.Second
)

type kelvinsResolverBuilder struct{}

func (*kelvinsResolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	ctx, cancel := context.WithCancel(context.Background())
	r := &kelvinsResolver{
		target: target,
		cc:     cc,
		rn:     make(chan struct{}, 1),
		ctx:    ctx,
		cancel: cancel,
	}

	go r.watcher()
	r.ResolveNow(resolver.ResolveNowOptions{})

	return r, nil
}

func (*kelvinsResolverBuilder) Scheme() string { return kelvinsScheme }

type kelvinsResolver struct {
	target resolver.Target
	cc     resolver.ClientConn
	rn     chan struct{}
	ctx    context.Context
	cancel context.CancelFunc
}

func (r *kelvinsResolver) watcher() {
	for {
		select {
		case <-kelvins.AppCloseCh:
			return
		case <-r.ctx.Done():
			return
		case <-r.rn:
		}

		// 执行解析
		r.resolverServiceConfig()

		// 休眠以防止过度重新解析。 传入的解决请求
		// 将在 d.rn 中排队。
		t := time.NewTimer(minResolverRate)
		select {
		case <-t.C:
		case <-kelvins.AppCloseCh:
			t.Stop()
			return
		case <-r.ctx.Done():
			t.Stop()
			return
		}
	}
}

var emptyCtx = context.Background()

func (r *kelvinsResolver) resolverServiceConfig() {
	serviceName := r.target.Endpoint
	etcdServerUrls := config.GetEtcdV3ServerURLs()
	serviceLB := slb.NewService(etcdServerUrls, serviceName)
	serviceConfigClient := etcdconfig.NewServiceConfigClient(serviceLB)
	var serviceConfigs map[string]*etcdconfig.Config
	var err error
	// 有限的重试
	for i := 0; i < 3; i++ {
		serviceConfigs, err = serviceConfigClient.GetConfigs()
		if err == nil {
			break
		}
	}
	if err != nil {
		r.cc.ReportError(fmt.Errorf("serviceConfigClient GetConfigs err: %v, key suffix: %v", err, serviceName))
		if vars.FrameworkLogger != nil {
			vars.FrameworkLogger.Errorf(emptyCtx, "serviceConfigClient GetConfigs err: %v, key suffix: %v", err, serviceName)
		} else {
			logging.Errf("serviceConfigClient GetConfigs err: %v, key suffix: %v\n", err, serviceName)
		}
		return
	}

	if len(serviceConfigs) == 0 {
		return
	}

	address := make([]resolver.Address, 0, len(serviceConfigs))
	for _, value := range serviceConfigs {
		addr := fmt.Sprintf("%v:%v", serviceName, value.ServicePort)
		address = append(address, resolver.Address{
			Addr: addr,
		})
	}
	if len(address) > 0 {
		r.cc.UpdateState(resolver.State{Addresses: address})
	}
}

func (r *kelvinsResolver) ResolveNow(o resolver.ResolveNowOptions) {
	select {
	case r.rn <- struct{}{}:
	default:
	}
}

func (r *kelvinsResolver) Close() { r.cancel() }

func init() {
	resolver.Register(&kelvinsResolverBuilder{})
}
