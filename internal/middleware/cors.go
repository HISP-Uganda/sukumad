package middleware

import (
	"slices"
	"strings"
	"sukumad/config"

	"github.com/gin-gonic/gin"
)

func CORSFromConfig(cfg config.Config) gin.HandlerFunc {
	allowed := cfg.AllowedOrigins
	all := len(allowed) == 0 || slices.Contains(allowed, "*")

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if all || matchOrigin(origin, allowed) {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Vary", "Origin")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func matchOrigin(origin string, list []string) bool {
	o := strings.TrimSpace(origin)
	for _, v := range list {
		if strings.EqualFold(strings.TrimSpace(v), o) {
			return true
		}
	}
	return false
}
