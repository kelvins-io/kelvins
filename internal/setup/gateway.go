package setup

import (
	"context"
	"gitee.com/kelvins-io/common/errcode"
	"gitee.com/kelvins-io/common/json"
	"gitee.com/kelvins-io/common/proto/common"
	"gitee.com/kelvins-io/kelvins/internal/vars"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
	"net/http"
)

type GRPCErrReturn struct {
	ErrCode   int32  `json:"code,omitempty"`   // 错误码
	ErrMsg    string `json:"error,omitempty"`  // 错误消息
	ErrDetail string `json:"detail,omitempty"` // 错误详情
}

// NewGateway ...
func NewGateway() *runtime.ServeMux {
	runtime.HTTPError = customHTTPError
	return runtime.NewServeMux()
}

// customHTTPError customs grpc-gateway response json.
func customHTTPError(ctx context.Context, _ *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
	s, ok := status.FromError(err)
	if !ok {
		s = status.New(codes.Unknown, err.Error())
	}

	grpcErrReturn := GRPCErrReturn{}
	details := s.Details()
	isDetail := false
	for _, detail := range details {
		if v, ok := detail.(*common.Error); ok {
			grpcErrReturn.ErrCode = v.Code
			grpcErrReturn.ErrMsg = v.Message
			isDetail = true
			break
		}
	}

	if isDetail == false && s.Message() != "" {
		errCode := errcode.FAIL
		if s.Code() == codes.DeadlineExceeded {
			errCode = errcode.DEADLINE_EXCEEDED
		}

		grpcErrReturn.ErrCode = int32(errCode)
		grpcErrReturn.ErrMsg = errcode.GetErrMsg(errCode)
		grpcErrReturn.ErrDetail = s.Message()

		if vars.ErrLogger != nil {
			vars.ErrLogger.Errorf(ctx, "grpc-gateway(%s) err: %s", r.RemoteAddr+":"+r.RequestURI, s.Message())
		} else {
			log.Printf("grpc-gateway(%s) err: %s\n", r.RemoteAddr+":"+r.RequestURI, s.Message())
		}
	}

	respMessage, _ := json.Marshal(grpcErrReturn)

	w.Header().Set("Content-type", marshaler.ContentType())
	w.WriteHeader(errcode.ToHttpStatusCode(s.Code()))
	_, err = w.Write(respMessage)
	if err != nil {
		if vars.ErrLogger != nil {
			vars.ErrLogger.Errorf(ctx, "Gateway(%s) response write err: %v, msg: %s", r.RemoteAddr+":"+r.RequestURI, err, s.Message())
		} else {
			log.Printf("Gateway(%s) response write err: %v, msg: %s\n", r.RemoteAddr+":"+r.RequestURI, err, s.Message())
		}
	}
}
