package setup

import (
	"gitee.com/kelvins-io/kelvins/internal/metrics_mux"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"net/http"
)

// NewServerMux ...
func NewServerMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux = metrics_mux.GetElasticMux(mux)
	mux = metrics_mux.GetPProfMux(mux)
	mux = metrics_mux.GetPrometheusMux(mux)
	return mux
}

// NewGatewayServerMux ...
func NewGatewayServerMux(gateway *runtime.ServeMux) *http.ServeMux {
	mux := NewServerMux()
	mux.Handle("/", gateway)
	return mux
}
