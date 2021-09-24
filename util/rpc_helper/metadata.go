package rpc_helper

import (
	"context"
	"fmt"
	"gitee.com/kelvins-io/kelvins"
	"gitee.com/kelvins-io/kelvins/internal/vars"
	"google.golang.org/grpc/metadata"
)

func GetRequestId(ctx context.Context) (requestId string) {
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		if t, ok := md[kelvins.RPCMetadataRequestId]; ok {
			for _, e := range t {
				if e != "" {
					requestId = e
					ok = true
					break
				}
			}
		}
	}
	return
}

type RequestMeta struct {
	RequestId string
	Version   string
}

func GetRequestMetadata(ctx context.Context) *RequestMeta {
	return &RequestMeta{
		RequestId: GetRequestId(ctx),
		Version:   fmt.Sprintf("kelvins/rpc %v", vars.Version),
	}
}
