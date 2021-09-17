package gin_helper

import (
	"github.com/gin-gonic/gin"
)

func JsonResponse(ctx *gin.Context, httpCode, retCode int, data interface{}) {
	ctx.JSON(httpCode, gin.H{
		"code": retCode,
		"msg":  GetMsg(retCode),
		"data": data,
	})
}

func ProtoBufResponse(ctx *gin.Context, httpCode int, data interface{}) {
	ctx.ProtoBuf(httpCode, data)
}
