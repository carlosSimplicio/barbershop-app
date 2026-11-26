package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

// DecodeJSONRequest accepts one bounded JSON document and refuses unknown fields.
// Domain handlers retain responsibility for presence, nullability and error codes.
func DecodeJSONRequest(httpContext *gin.Context, input any, maxBytes int64) error {
	decoder := json.NewDecoder(http.MaxBytesReader(httpContext.Writer, httpContext.Request.Body, maxBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(input); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("expected a single JSON document")
	}
	return nil
}
