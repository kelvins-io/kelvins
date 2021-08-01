package client_conn

import (
	"context"
	"fmt"
	"gitee.com/kelvins-io/kelvins"
	"gitee.com/kelvins-io/kelvins/internal/config"
	"gitee.com/kelvins-io/kelvins/internal/service/slb"
	"gitee.com/kelvins-io/kelvins/internal/service/slb/etcdconfig"
	"google.golang.org/grpc/resolver"
)

const (
	kelvinsScheme = "kelvins-scheme"
)

type kelvinsResolverBuilder struct{}

func (*kelvinsResolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	r := &kelvinsResolver{
		target: target,
		cc:     cc,
	}

	r.watchServiceConfig()

	return r, nil
}

func (*kelvinsResolverBuilder) Scheme() string { return kelvinsScheme }

type kelvinsResolver struct {
	target resolver.Target
	cc     resolver.ClientConn
}

//func (r *kelvinsResolver) start() {
//	ticker := time.NewTicker(3 * time.Second)
//	for {
//		select {
//		case <-kelvins.AppCloseCh:
//			ticker.Stop()
//			return
//		case <-ticker.C:
//			r.watchServiceConfig()
//		}
//	}
//}

func (r *kelvinsResolver) watchServiceConfig() {
	serviceName := r.target.Endpoint
	etcdServerUrls := config.GetEtcdV3ServerURLs()
	serviceLB := slb.NewService(etcdServerUrls, serviceName)
	serviceConfigClient := etcdconfig.NewServiceConfigClient(serviceLB)
	serviceConfigs, err := serviceConfigClient.GetConfigs()
	if err != nil {
		r.cc.ReportError(err)
		if kelvins.FrameworkLogger != nil {
			kelvins.FrameworkLogger.Errorf(context.Background(), "@@watchServiceConfig GetConfigs err: %v, serviceName: %v", err, serviceName)
		} else if kelvins.AccessLogger != nil {
			kelvins.AccessLogger.Errorf(context.Background(), "@@watchServiceConfig GetConfigs err: %v, serviceName: %v", err, serviceName)
		}

		return
	}

	address := make([]resolver.Address, 0, len(serviceConfigs))
	for _, value := range serviceConfigs {
		addr := fmt.Sprintf("%v:%v", serviceName, value.ServicePort)
		address = append(address, resolver.Address{
			Addr:       addr,
			ServerName: serviceName,
		})
	}

	r.cc.UpdateState(resolver.State{Addresses: address})
	ctx := context.Background()
	if kelvins.FrameworkLogger != nil {
		kelvins.FrameworkLogger.Infof(ctx, "@@kelvinsResolver watchServiceConfig UpdateState serviceName(%v), address: %+v", serviceName, address)
	} else if kelvins.AccessLogger != nil {
		kelvins.AccessLogger.Infof(ctx, "@@kelvinsResolver watchServiceConfig UpdateState serviceName(%v), address: %+v", serviceName, address)
	}
}

func (r *kelvinsResolver) ResolveNow(o resolver.ResolveNowOptions) {
	ctx := context.Background()
	if kelvins.FrameworkLogger != nil {
		kelvins.FrameworkLogger.Infof(ctx, "@@kelvinsResolver ResolveNow ")
	} else if kelvins.AccessLogger != nil {
		kelvins.AccessLogger.Infof(ctx, "@@kelvinsResolver ResolveNow ")
	}

	r.watchServiceConfig()
}
func (*kelvinsResolver) Close() {}

func init() {
	resolver.Register(&kelvinsResolverBuilder{})
}
