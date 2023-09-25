package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/serg666/ecom-service/httputil"
)

type healthCheckHandler struct {
	limiter *httputil.Limiter
}

func (h healthCheckHandler) Health(ctx *gin.Context) {
	if !h.limiter.Allow() {
		ctx.JSON(http.StatusTooManyRequests, gin.H{
			"error": http.StatusText(http.StatusTooManyRequests),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"health": true,
	})
}

func NewHealthCheckHandler() *healthCheckHandler {
	return &healthCheckHandler{
		limiter: httputil.NewLimiter(3, 2*time.Minute),
	}
}
