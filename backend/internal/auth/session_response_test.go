package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSessionCookieKeepsCSRFForFifteenDays(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	setSessionCookies(c, "access", "csrf")

	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == csrfCookieName {
			if cookie.MaxAge != 15*24*60*60 || cookie.Path != "/" || !cookie.Secure || cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode {
				t.Fatalf("unexpected CSRF cookie properties: max_age=%d path=%q secure=%v http_only=%v same_site=%d", cookie.MaxAge, cookie.Path, cookie.Secure, cookie.HttpOnly, cookie.SameSite)
			}
			return
		}
	}
	t.Fatal("CSRF cookie not set")
}
