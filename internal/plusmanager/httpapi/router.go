package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Options struct {
	Enabled bool
}

func RegisterRoutes(group *gin.RouterGroup, opts Options) {
	if group == nil || !opts.Enabled {
		return
	}

	group.GET("/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"mode":   "integrated",
		})
	})
	group.GET("/info", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"integrated":    true,
			"setupRequired": false,
		})
	})
}
