package setup

import (
	"google.golang.org/grpc"
)

// NewGRPC ...
func NewGRPC(opts []grpc.ServerOption) *grpc.Server {
	return grpc.NewServer(opts...)
}
