package httpserver

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"regexp"
	"time"

	api "barbershop-app/backend/openapi/generated"

	"github.com/gin-gonic/gin"
)

var validRequestID = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

func RequestContext() gin.HandlerFunc {
	return func(httpContext *gin.Context) {
		requestID := httpContext.GetHeader("X-Request-ID")
		if !validRequestID.MatchString(requestID) {
			requestID = newRequestID()
		}

		httpContext.Set("request_id", requestID)
		httpContext.Header("X-Request-ID", requestID)
		httpContext.Next()
	}
}

func AccessLog(logger *slog.Logger) gin.HandlerFunc {
	return func(httpContext *gin.Context) {
		startedAt := time.Now()
		httpContext.Next()

		route := httpContext.FullPath()
		if route == "" {
			route = "unmatched"
		}
		logger.Info("http_request",
			"method", httpContext.Request.Method,
			"route", route,
			"status", httpContext.Writer.Status(),
			"duration_ms", time.Since(startedAt).Milliseconds(),
			"request_id", RequestID(httpContext),
		)
	}
}

func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return func(httpContext *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Error("http_panic", "request_id", RequestID(httpContext))
				httpContext.AbortWithStatusJSON(http.StatusInternalServerError, api.ErrorResponse{
					Code:      "internal_error",
					Message:   "Não foi possível concluir a solicitação.",
					RequestId: RequestID(httpContext),
				})
			}
		}()
		httpContext.Next()
	}
}

func RequestID(httpContext *gin.Context) string {
	value, _ := httpContext.Get("request_id")
	requestID, _ := value.(string)
	return requestID
}

func newRequestID() string {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(buffer)
}
