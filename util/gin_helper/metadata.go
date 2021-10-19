package gin_helper

import (
	"fmt"
	"gitee.com/kelvins-io/kelvins"
	"gitee.com/kelvins-io/kelvins/internal/vars"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"os"
	"time"
)

const (
	startTimeKey = "income-time"
)

func Metadata(debug bool) gin.HandlerFunc {
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
		if debug {
			c.Header(kelvins.HttpMetadataServiceNode, getRPCNodeInfo())
		}

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

var (
	hostName, _ = os.Hostname()
)

func getRPCNodeInfo() (nodeInfo string) {
	nodeInfo = fmt.Sprintf("%v:%v(%v)", vars.ServiceIp, vars.ServicePort, hostName)
	return
}
