package slb

type ServiceLB struct {
	EtcdServerUrl string
	ServerName    string
}

// NewService return ServiceLB instance.
func NewService(etcdServerUrl, serverName string) *ServiceLB {
	return &ServiceLB{EtcdServerUrl: etcdServerUrl, ServerName: serverName}
}
