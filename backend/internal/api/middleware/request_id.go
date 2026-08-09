package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	RequestIDKey    = "requestId"
	RequestIDHeader = "X-Request-ID"
)

// RequestID generates a unique UUID per request and attaches it to context and response headers.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader(RequestIDHeader)
		if reqID == "" {
			reqID = uuid.New().String()
		}

		c.Set(RequestIDKey, reqID)
		c.Header(RequestIDHeader, reqID)

		c.Next()
	}
}

// GetRequestID retrieves the requestId from Gin context.
func GetRequestID(c *gin.Context) string {
	if val, exists := c.Get(RequestIDKey); exists {
		if id, ok := val.(string); ok && id != "" {
			return id
		}
	}
	return ""
}
