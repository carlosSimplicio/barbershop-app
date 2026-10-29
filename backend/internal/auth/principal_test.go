package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	api "barbershop-app/backend/openapi/generated"

	"github.com/gin-gonic/gin"
)

func TestRequirePrincipalRejectsMissingIdentity(t *testing.T) {
	response := httptest.NewRecorder()
	httpContext, _ := gin.CreateTestContext(response)
	_, ok := RequirePrincipal(httpContext)
	if ok || !httpContext.IsAborted() || response.Code != http.StatusUnauthorized {
		t.Fatalf("missing identity: ok=%v aborted=%v status=%d", ok, httpContext.IsAborted(), response.Code)
	}
	var failure api.ErrorResponse
	if err := json.Unmarshal(response.Body.Bytes(), &failure); err != nil || failure.Code != "authentication_required" {
		t.Fatalf("unexpected authentication response: code=%s err=%v", failure.Code, err)
	}
}

func TestRequirePrincipalPreservesAuthenticatedIdentity(t *testing.T) {
	httpContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	expected := Principal{UserID: "owner", EstablishmentID: "tenant", Role: "owner"}
	httpContext.Set(principalContextKey, expected)
	principal, ok := RequirePrincipal(httpContext)
	if !ok || principal != expected || httpContext.IsAborted() {
		t.Fatalf("authenticated identity: principal=%+v ok=%v aborted=%v", principal, ok, httpContext.IsAborted())
	}
}
