//go:build integration

package integration_test

import (
	"net/http"
	"testing"
)

func TestHealthReportsReadyDatabase(t *testing.T) {
	application := newTestApplication(t)
	response := application.request(t, http.MethodGet, "/api/v1/health", nil)
	assertStatus(t, response, http.StatusOK)
}
