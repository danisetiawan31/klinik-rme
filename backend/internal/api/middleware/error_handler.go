package middleware

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
)

type ErrorResponseBody struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"requestId"`
}

type ErrorEnvelope struct {
	Error ErrorResponseBody `json:"error"`
}

// RespondError logs the raw error server-side with requestId, reports 5xx errors to Sentry, and returns curated JSON to client.
func RespondError(c *gin.Context, httpStatus int, code string, message string, rawErr error) {
	reqID := GetRequestID(c)

	if rawErr != nil {
		log.Printf("[ERROR] [requestId=%s] [status=%d] [code=%s] [rawErr=%v]", reqID, httpStatus, code, rawErr)

		// Capture server-side 5xx errors to Sentry with contextual tags
		if httpStatus >= 500 {
			sentry.WithScope(func(scope *sentry.Scope) {
				scope.SetTag("requestId", reqID)
				scope.SetTag("code", code)
				scope.SetTag("status", fmt.Sprintf("%d", httpStatus))
				sentry.CaptureException(rawErr)
			})
		}
	} else {
		log.Printf("[ERROR] [requestId=%s] [status=%d] [code=%s] [message=%s]", reqID, httpStatus, code, message)
	}

	c.JSON(httpStatus, ErrorEnvelope{
		Error: ErrorResponseBody{
			Code:      code,
			Message:   message,
			RequestID: reqID,
		},
	})
}

// GlobalRecovery catches panics, captures them to Sentry, logs stack traces, and returns a formatted 500 error response.
func GlobalRecovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				reqID := GetRequestID(c)
				stackTrace := string(debug.Stack())
				log.Printf("[PANIC] [requestId=%s] panic: %v\nStack trace:\n%s", reqID, err, stackTrace)

				// Report panic exception to Sentry
				sentry.WithScope(func(scope *sentry.Scope) {
					scope.SetTag("requestId", reqID)
					scope.SetTag("code", "INTERNAL_SERVER_ERROR")
					scope.SetContext("debug", map[string]interface{}{
						"stackTrace": stackTrace,
					})
					sentry.CaptureException(fmt.Errorf("panic: %v", err))
				})

				RespondError(
					c,
					http.StatusInternalServerError,
					"INTERNAL_SERVER_ERROR",
					"Terjadi kesalahan internal pada server",
					fmt.Errorf("%v", err),
				)
				c.Abort()
			}
		}()
		c.Next()
	}
}
