package setup

import (
	"gitee.com/kelvins-io/kelvins/internal/metrics_mux"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"net/http"
)

// NewServerMux ...
func NewServerMux(isMonitor bool) *http.ServeMux {
	mux := http.NewServeMux()
	if !isMonitor {
		return mux
	}
	mux = metrics_mux.GetElasticMux(mux)
	mux = metrics_mux.GetPProfMux(mux)
	mux = metrics_mux.GetPrometheusMux(mux)
	return mux
}

// NewGatewayServerMux ...
func NewGatewayServerMux(gateway *runtime.ServeMux, isMonitor bool) *http.ServeMux {
	mux := NewServerMux(isMonitor)
	mux.Handle("/", gateway)
	return mux
}
