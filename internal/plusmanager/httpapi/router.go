package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/plusmanager/store"
)

type ModelPriceStore interface {
	ListModelPrices() ([]store.ModelPrice, error)
	ReplaceModelPrices([]store.ModelPrice) error
}

type Options struct {
	Enabled bool
	Store   ModelPriceStore
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
	group.GET("/model-prices", func(c *gin.Context) {
		if opts.Store == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "model price store unavailable"})
			return
		}
		prices, err := opts.Store.ListModelPrices()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "list model prices failed"})
			return
		}
		c.JSON(http.StatusOK, prices)
	})
	group.PUT("/model-prices", func(c *gin.Context) {
		var prices []store.ModelPrice
		if err := c.ShouldBindJSON(&prices); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
			return
		}
		if opts.Store == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "model price store unavailable"})
			return
		}
		if err := opts.Store.ReplaceModelPrices(prices); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "replace model prices failed"})
			return
		}
		c.Status(http.StatusNoContent)
	})
}
