//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestProfessionalMutationsRejectInvalidJSONWithoutWriting(t *testing.T) {
	app := newTestApplication(t)
	id := createManagementProfessional(t, app)
	deactivateManagementProfessional(t, app, id)
	before := readProfessionalSecurityState(t, id)
	for _, operation := range []struct{ name, method, path, body string }{
		{"create", http.MethodPost, "/api/v1/professionals", `{"name":"Bia","email":"bia@example.test","temporary_password":"senha-temporaria"}`},
		{"update", http.MethodPatch, "/api/v1/professionals/" + id, `{"name":"Bia"}`},
		{"reactivate", http.MethodPost, "/api/v1/professionals/" + id + "/reactivation", `{"temporary_password":"outra-senha"}`},
	} {
		for _, invalid := range []struct{ name, body string }{
			{"unknown field", strings.TrimSuffix(operation.body, "}") + `,"unexpected":true}`},
			{"extra document", operation.body + ` {}`},
			{"over 8 KiB", operation.body + strings.Repeat(" ", 8192)},
			{"truncated", strings.TrimSuffix(operation.body, "}")},
		} {
			t.Run(operation.name+"/"+invalid.name, func(t *testing.T) {
				request, err := http.NewRequestWithContext(context.Background(), operation.method, app.server.URL+operation.path, strings.NewReader(invalid.body))
				if err != nil {
					t.Fatal(err)
				}
				request.Header.Set("Origin", trustedOrigin)
				request.Header.Set("Content-Type", "application/json")
				request.Header.Set("X-CSRF-Token", app.cookie(t, csrfCookieName).Value)
				response, err := app.client.Do(request)
				if err != nil {
					t.Fatal(err)
				}
				defer response.Body.Close()
				assertStatus(t, response, http.StatusBadRequest)
				var failure struct{ Code string }
				decodeJSON(t, response, &failure)
				if failure.Code != "invalid_request" {
					t.Fatal("invalid JSON did not receive invalid_request")
				}
			})
		}
	}
	if readProfessionalSecurityState(t, id) != before {
		t.Fatal("invalid request changed professional data or credentials")
	}
	if len(getProfessionalPage(t, app, "/api/v1/professionals").Items) != 1 {
		t.Fatal("invalid create persisted a professional")
	}
}
