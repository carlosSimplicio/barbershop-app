package catalog

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	api "barbershop-app/backend/openapi/generated"

	"github.com/gin-gonic/gin"
)

func TestCatalogValidationErrorsPresentBusinessLimits(t *testing.T) {
	for _, test := range []struct {
		name    string
		err     error
		field   string
		code    string
		message string
	}{
		{"name", ErrInvalidName, "name", "invalid_name", "Informe um nome com até 120 caracteres."},
		{"price", ErrInvalidPrice, "price_cents", "invalid_price", "Informe um preço de R$ 0,00 a R$ 99.999,99."},
		{"duration", ErrInvalidDuration, "duration_minutes", "invalid_duration", "Informe de 1 a 1.440 minutos inteiros."},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			httpContext, _ := gin.CreateTestContext(response)
			writeCatalogError(httpContext, test.err)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status=%d want=%d", response.Code, http.StatusBadRequest)
			}
			var failure api.ErrorResponse
			if err := json.Unmarshal(response.Body.Bytes(), &failure); err != nil {
				t.Fatal(err)
			}
			if failure.Code != "validation_failed" || failure.Fields == nil || len(*failure.Fields) != 1 {
				t.Fatalf("unexpected field error: %+v", failure)
			}
			field := (*failure.Fields)[0]
			if field.Field != test.field || field.Code != test.code || field.Message != test.message {
				t.Fatalf("unexpected field error: %+v", field)
			}
		})
	}
}
