package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type databaseStub struct {
	err error
}

func (stub databaseStub) Ping(context.Context) error { return stub.err }

func TestGetHealthReturnsServiceUnavailableWhenDatabaseIsDown(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	context.Set("request_id", "request-123")

	NewHandler(databaseStub{err: errors.New("database unavailable")}).GetHealth(context)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
	if !strings.Contains(recorder.Body.String(), `"request_id":"request-123"`) {
		t.Errorf("response does not contain request ID: %s", recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "database unavailable") {
		t.Errorf("response leaks internal error: %s", recorder.Body.String())
	}
}
