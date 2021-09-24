package setup

import (
	"crypto/tls"
	"gitee.com/kelvins-io/kelvins/config/setting"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"net/http"
	"strings"
)

// NewHttpServer ...
func NewHttpServer(handler http.Handler, tlsConfig *tls.Config, serverSetting *setting.HttpServerSettingS) *http.Server {
	return &http.Server{
		Handler:      handler,
		TLSConfig:    tlsConfig,
		Addr:         serverSetting.GetAddr(),
		ReadTimeout:  serverSetting.GetReadTimeout(),
		WriteTimeout: serverSetting.GetWriteTimeout(),
		IdleTimeout:  serverSetting.GetIdleTimeout(),
	}
}

func NewHttp2Server(handler http.Handler, tlsConfig *tls.Config, serverSetting *setting.HttpServerSettingS) *http.Server {
	return NewHttpServer(h2c.NewHandler(handler, &http2.Server{IdleTimeout: serverSetting.GetIdleTimeout()}), tlsConfig, serverSetting)
}

// GRPCHandlerFunc gRPCHandlerFunc ...
func GRPCHandlerFunc(grpcServer *grpc.Server, otherHandler http.Handler, serverSetting *setting.HttpServerSettingS) http.Handler {
	if otherHandler == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			grpcServer.ServeHTTP(w, r)
		})
	}
	return h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	}), &http2.Server{
		IdleTimeout: serverSetting.GetIdleTimeout(),
	})
}
