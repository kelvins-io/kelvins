package rpc_helper

import (
	"context"
	"gitee.com/kelvins-io/kelvins"
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
