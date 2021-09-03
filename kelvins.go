package kelvins

import (
	"context"
	"crypto/tls"
	"gitee.com/kelvins-io/common/event"
	"gitee.com/kelvins-io/common/log"
	"gitee.com/kelvins-io/common/queue"
	"github.com/gin-gonic/gin"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/robfig/cron/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"net/http"
)

const (
	AppTypeGrpc  = 1
	AppTypeCron  = 2
	AppTypeQueue = 3
	AppTypeHttp  = 4
)

var (
	AppTypeText = map[int32]string{
		AppTypeGrpc:  "gRPC",
		AppTypeCron:  "Cron",
		AppTypeQueue: "Queue",
		AppTypeHttp:  "Http",
	}
)

// Application ...
type Application struct {
	Name           string
	Type           int32
	LoggerRootPath string
	LoggerLevel    string
	Environment    string
	LoadConfig     func() error
	SetupVars      func() error
	StopFunc       func() error
}

// GRPCApplication ...
type GRPCApplication struct {
	*Application
	Port                     int64
	GRPCServer               *grpc.Server
	HealthServer             *GRPCHealthServer
	DisableHealthCheck       bool
	RegisterHealthServer     func(*GRPCHealthServer) // execute in the coroutine
	NumServerWorkers         uint32
	GatewayServeMux          *runtime.ServeMux
	Mux                      *http.ServeMux
	HttpServer               *http.Server
	TlsConfig                *tls.Config
	GKelvinsLogger           log.LoggerContextIface
	GSysErrLogger            log.LoggerContextIface
	UnaryServerInterceptors  []grpc.UnaryServerInterceptor
	StreamServerInterceptors []grpc.StreamServerInterceptor
	ServerOptions            []grpc.ServerOption
	RegisterGRPCServer       func(*grpc.Server) error
	RegisterGateway          func(context.Context, *runtime.ServeMux, string, []grpc.DialOption) error
	RegisterHttpRoute        func(*http.ServeMux) error
	EventServer              *event.EventServer
	RegisterEventProducer    func(event.ProducerIface) error
}

type GRPCHealthServer struct {
	*health.Server
}

// AuthFuncOverride let go of health check
func (a *GRPCHealthServer) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, nil
}

// CronJob warps job define.
type CronJob struct {
	Name string // Job unique name
	Spec string // Job specification
	Job  func() // Job func
}

// CronApplication ...
type CronApplication struct {
	*Application
	CronLogger           log.LoggerContextIface
	Cron                 *cron.Cron
	GenCronJobs          func() []*CronJob
	EventServer          *event.EventServer
	RegisterEventHandler func(event.EventServerIface) error
}

// QueueApplication ...
type QueueApplication struct {
	*Application
	QueueLogger          log.LoggerContextIface
	QueueServer          *queue.MachineryQueue
	EventServer          *event.EventServer
	GetNamedTaskFuncs    func() map[string]interface{}
	RegisterEventHandler func(event.EventServerIface) error
}

// HTTPApplication ...
type HTTPApplication struct {
	*Application
	Port                  int64
	TraceLogger           log.LoggerContextIface
	TlsConfig             *tls.Config
	Mux                   *http.ServeMux
	HttpServer            *http.Server
	RegisterHttpRoute     func(*http.ServeMux) error
	RegisterHttpGinEngine func() (*gin.Engine, error) // is not nil will over RegisterHttpRoute
	EventServer           *event.EventServer
	RegisterEventProducer func(event.ProducerIface) error
}
