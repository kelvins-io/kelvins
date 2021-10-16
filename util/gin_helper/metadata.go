package gin_helper

import (
	"fmt"
	"gitee.com/kelvins-io/kelvins"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net"
	"os"
	"strings"
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
	hostName, _   = os.Hostname()
	outBoundIP, _ = getOutBoundIP()
)

func getRPCNodeInfo() (nodeInfo string) {
	nodeInfo = fmt.Sprintf("%v(%v)", outBoundIP, hostName)
	return
}

func getOutBoundIP() (ip string, err error) {
	// broadcast
	conn, err := net.Dial("udp", "255.255.255.255:53")
	if err != nil {
		return
	}
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	ip = strings.Split(localAddr.String(), ":")[0]
	return
}
