package offerings

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAssociationBodyRequiresOneBoundedBoolean(t *testing.T) {
	for _, body := range []string{``, `{}`, `null`, `{"associated":null}`, `{"associated":1}`, `{"associated":"true"}`, `{"associated":[]}`, `{"associated":true,"extra":1}`, `{"associated":true} {}`, `{"associated":`, strings.Repeat(" ", 8192) + `{"associated":true}`} {
		t.Run(body[:min(len(body), 40)], func(t *testing.T) {
			httpContext, _ := gin.CreateTestContext(httptest.NewRecorder())
			httpContext.Request = httptest.NewRequest("PUT", "/", strings.NewReader(body))
			if _, err := decodeAssociationRequest(httpContext); !errors.Is(err, ErrInvalidRequest) {
				t.Fatalf("body accepted, error=%v", err)
			}
		})
	}
	for _, body := range []string{`{"associated":true}`, `{"associated":false}`} {
		httpContext, _ := gin.CreateTestContext(httptest.NewRecorder())
		httpContext.Request = httptest.NewRequest("PUT", "/", strings.NewReader(body))
		value, err := decodeAssociationRequest(httpContext)
		if err != nil || value == nil || *value != strings.Contains(body, "true") {
			t.Fatalf("boolean=%v err=%v", value, err)
		}
	}
}
