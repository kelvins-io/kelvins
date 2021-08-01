package client_conn

import (
	"fmt"
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
		r.cc.ReportError(fmt.Errorf("serviceConfigClient GetConfigs err: %v, key suffix: %v", err, serviceName))
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
	if len(address) > 0 {
		r.cc.UpdateState(resolver.State{Addresses: address})
	}
}

func (r *kelvinsResolver) ResolveNow(o resolver.ResolveNowOptions) {
	r.watchServiceConfig()
}

func (*kelvinsResolver) Close() {}

func init() {
	resolver.Register(&kelvinsResolverBuilder{})
}
