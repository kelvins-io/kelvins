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
	"net/http"
)

const (
	AppTypeGrpc  = 1
	AppTypeCron  = 2
	AppTypeQueue = 3
	AppTypeHttp  = 4
)

// Application ...
type Application struct {
	Name           string
	Type           int32
	LoggerRootPath string
	LoggerLevel    string
	LoadConfig     func() error
	SetupVars      func() error
	StopFunc       func() error
}

// GRPCApplication ...
type GRPCApplication struct {
	*Application
	Port                    int64
	GRPCServer              *grpc.Server
	GatewayServeMux         *runtime.ServeMux
	Mux                     *http.ServeMux
	HttpServer              *http.Server
	TlsConfig               *tls.Config
	GKelvinsLogger          *log.LoggerContext
	GSysErrLogger           *log.LoggerContext
	UnaryServerInterceptors []grpc.UnaryServerInterceptor
	ServerOptions           []grpc.ServerOption
	RegisterGRPCServer      func(*grpc.Server) error
	RegisterGateway         func(context.Context, *runtime.ServeMux, string, []grpc.DialOption) error
	RegisterHttpRoute       func(*http.ServeMux) error
	EventServer             *event.EventServer
	RegisterEventProducer   func(event.ProducerIface) error
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
	CronLogger           *log.LoggerContext
	Cron                 *cron.Cron
	GenCronJobs          func() []*CronJob
	EventServer          *event.EventServer
	RegisterEventHandler func(event.EventServerIface) error
}

// QueueApplication ...
type QueueApplication struct {
	*Application
	QueueLogger          *log.LoggerContext
	QueueServer          *queue.MachineryQueue
	EventServer          *event.EventServer
	GetNamedTaskFuncs    func() map[string]interface{}
	RegisterEventHandler func(event.EventServerIface) error
}

// HTTPApplication ...
type HTTPApplication struct {
	*Application
	Port                  int64
	TraceLogger           *log.LoggerContext
	TlsConfig             *tls.Config
	Mux                   *http.ServeMux
	HttpServer            *http.Server
	RegisterHttpRoute     func(*http.ServeMux) error
	RegisterHttpGinEngine func() (*gin.Engine,error) // can over RegisterHttpRoute
	EventServer           *event.EventServer
	RegisterEventProducer func(event.ProducerIface) error
}
