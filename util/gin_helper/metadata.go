package gin_helper

import (
	"gitee.com/kelvins-io/kelvins"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"time"
)

const (
	startTimeKey = "income-time"
)

func Metadata() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		c.Set(startTimeKey, startTime)
		requestId := c.Request.Header.Get(kelvins.HttpMetadataRequestId)
		if requestId == "" {
			requestId = uuid.New().String()
		}
		c.Header(kelvins.HttpMetadataRequestId, requestId)
		c.Set(kelvins.HttpMetadataRequestId, requestId)
		c.Header(kelvins.HttpMetadataPowerBy, "kelvins/http(gin) "+kelvins.Version)
		c.Header(kelvins.HttpMetadataServiceName, kelvins.AppName)

		c.Next()
		// next after set header is invalid because the header must be sent before sending the body
	}
}

func GetRequestId(ctx *gin.Context) (requestId string) {
	v, ok := ctx.Get(kelvins.HttpMetadataRequestId)
	if ok {
		requestId, _ = v.(string)
	}
	return
}
