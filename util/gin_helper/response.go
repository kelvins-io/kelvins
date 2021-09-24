package gin_helper

import (
	"fmt"
	"gitee.com/kelvins-io/kelvins"
	"github.com/gin-gonic/gin"
	"time"
)

func JsonResponse(ctx *gin.Context, httpCode, retCode int, data interface{}) {
	echoStatistics(ctx)
	ctx.JSON(httpCode, gin.H{
		"code": retCode,
		"msg":  GetMsg(retCode),
		"data": data,
	})
}

func ProtoBufResponse(ctx *gin.Context, httpCode int, data interface{}) {
	echoStatistics(ctx)
	ctx.ProtoBuf(httpCode, data)
}

func echoStatistics(ctx *gin.Context) {
	startTimeVal, ok := ctx.Get(startTimeKey)
	if ok {
		startTime, ok := startTimeVal.(time.Time)
		if !ok {
			startTime = time.Time{}
		}
		endTime := time.Now()
		ctx.Header(kelvins.HttpMetadataHandleTime, fmt.Sprintf("%f/s", endTime.Sub(startTime).Seconds()))
		ctx.Header(kelvins.HttpMetadataResponseTime, endTime.Format(kelvins.ResponseTimeLayout))
	}
}
