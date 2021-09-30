package gin_helper

import (
	"gitee.com/kelvins-io/kelvins/util/middleware"
	"github.com/gin-gonic/gin"
	"net/http"
)

func RateLimit(maxConcurrent int) gin.HandlerFunc {
	var limiter middleware.Limiter
	if maxConcurrent > 0 {
		limiter = middleware.NewKelvinsRateLimit(maxConcurrent)
	}
	return func(c *gin.Context) {
		if limiter != nil {
			if limiter.Limit() {
				JsonResponse(c, http.StatusTooManyRequests, TooManyRequests, GetMsg(TooManyRequests))
				c.Abort()
				return
			}
			defer limiter.ReturnTicket()
		}
		c.Next()
	}
}
