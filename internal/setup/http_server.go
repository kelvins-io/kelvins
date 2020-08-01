package setup

import (
	"crypto/tls"
	"gitee.com/kelvins-io/kelvins/config/setting"
	"google.golang.org/grpc"
	"net/http"
	"strings"
)

// NewHttpServer ...
func NewHttpServer(handler http.Handler, tlsConfig *tls.Config, serverSetting *setting.ServerSettingS) *http.Server {
	return &http.Server{
		Addr:      serverSetting.EndPoint,
		Handler:   handler,
		TLSConfig: tlsConfig,
	}
}

// gRPCHandlerFunc ...
func GRPCHandlerFunc(grpcServer *grpc.Server, otherHandler http.Handler) http.Handler {
	if otherHandler == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			grpcServer.ServeHTTP(w, r)
		})
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			if r.Method == "OPTIONS" && r.Header.Get("Access-Control-Request-Method") != "" {
				// CORS
				w.Header().Set("Access-Control-Allow-Origin", "*")
				headers := []string{"Content-Type", "Accept"}
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(headers, ","))
				methods := []string{"GET", "HEAD", "POST", "PUT", "DELETE"}
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(methods, ","))
				return
			}
			otherHandler.ServeHTTP(w, r)
		}
	})
}
