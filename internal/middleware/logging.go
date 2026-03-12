package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

const requestIDHeader = "X-Request-ID"

// RequestLogger is a Gin middleware that emits structured logs for every request.
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Ensure/propagate request id
		rid := c.GetHeader(requestIDHeader)
		if rid == "" {
			rid = newReqID()
		}
		c.Writer.Header().Set(requestIDHeader, rid)
		c.Set(requestIDHeader, rid)

		start := time.Now()
		path := c.Request.URL.Path
		rawQuery := c.Request.URL.RawQuery
		method := c.Request.Method
		clientIP := c.ClientIP()
		ua := c.GetHeader("User-Agent")
		cl := c.Request.ContentLength

		// Process request
		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		respSize := c.Writer.Size()

		entry := log.WithFields(log.Fields{
			"request_id": rid,
			"method":     method,
			"path":       path,
			"query":      rawQuery,
			"status":     status,
			"latency_ms": latency.Milliseconds(),
			"ip":         clientIP,
			"user_agent": ua,
			"req_bytes":  cl,
			"resp_bytes": respSize,
		})

		// If Gin recorded errors, log them
		if len(c.Errors) > 0 {
			entry.WithField("errors", c.Errors.String()).Error("http_request")
			return
		}

		// Level by status class
		switch {
		case status >= 500:
			entry.Error("http_request")
		case status >= 400:
			entry.Warn("http_request")
		default:
			entry.Info("http_request")
		}
	}
}

func newReqID() string {
	var b [12]byte // 24 hex chars
	if _, err := rand.Read(b[:]); err != nil {
		return "rid-" + time.Now().Format("20060102150405.000")
	}
	return hex.EncodeToString(b[:])
}
