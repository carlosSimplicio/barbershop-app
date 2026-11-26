//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

type professionalPairState struct {
	ID                   string
	Active               bool
	CreatedAt, UpdatedAt time.Time
}

func readProfessionalPair(t *testing.T, professionalID, offeringID string) professionalPairState {
	t.Helper()
	var state professionalPairState
	if err := integrationPool.QueryRow(context.Background(), `SELECT id,active,created_at,updated_at FROM professional_services WHERE professional_id=$1 AND service_id=$2`, professionalID, offeringID).Scan(&state.ID, &state.Active, &state.CreatedAt, &state.UpdatedAt); err != nil {
		t.Fatal(err)
	}
	return state
}
func setProfessionalPair(t *testing.T, app *testApplication, professionalID, offeringID string, associated bool) professionalOfferingRecord {
	t.Helper()
	response := app.request(t, http.MethodPut, "/api/v1/professionals/"+professionalID+"/services/"+offeringID, map[string]bool{"associated": associated})
	assertStatus(t, response, http.StatusOK)
	var record professionalOfferingRecord
	decodeJSON(t, response, &record)
	if record.ID != offeringID || record.Associated != associated {
		t.Fatalf("wrong confirmation=%+v", record)
	}
	return record
}
func TestProfessionalServicesMutationIsIdempotentAndPreservesCatalogAndCredentials(t *testing.T) {
	app := newTestApplication(t)
	id := createManagementProfessional(t, app)
	offering := createCatalogOffering(t, app, "Corte", 1234)
	before := readProfessionalSecurityState(t, id)
	record := setProfessionalPair(t, app, id, offering.ID, false)
	if record.offeringRecord != offering {
		t.Fatal("removal changed catalog")
	}
	var count int
	if err := integrationPool.QueryRow(context.Background(), `SELECT count(*) FROM professional_services`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("absent removal created pair: %d %v", count, err)
	}
	setProfessionalPair(t, app, id, offering.ID, true)
	original := readProfessionalPair(t, id, offering.ID)
	setProfessionalPair(t, app, id, offering.ID, true)
	if readProfessionalPair(t, id, offering.ID) != original {
		t.Fatal("repeated association changed identity/dates")
	}
	setProfessionalPair(t, app, id, offering.ID, false)
	removed := readProfessionalPair(t, id, offering.ID)
	if removed.ID != original.ID || removed.Active || removed.CreatedAt != original.CreatedAt || !removed.UpdatedAt.After(original.UpdatedAt) {
		t.Fatalf("removal did not preserve pair: %+v %+v", original, removed)
	}
	setProfessionalPair(t, app, id, offering.ID, false)
	if readProfessionalPair(t, id, offering.ID) != removed {
		t.Fatal("repeated removal changed dates")
	}
	setProfessionalPair(t, app, id, offering.ID, true)
	restored := readProfessionalPair(t, id, offering.ID)
	if restored.ID != original.ID || restored.CreatedAt != original.CreatedAt || !restored.Active {
		t.Fatal("reassociation replaced pair")
	}
	if readProfessionalSecurityState(t, id) != before {
		t.Fatal("association changed professional or credentials")
	}
	assertStatus(t, app.request(t, http.MethodPut, "/api/v1/services/"+offering.ID+"/status", map[string]bool{"active": false}), http.StatusOK)
	deactivateManagementProfessional(t, app, id)
	if readProfessionalPair(t, id, offering.ID) != restored {
		t.Fatal("deactivation changed pair")
	}
	inactiveBefore := readProfessionalSecurityState(t, id)
	inactive := setProfessionalPair(t, app, id, offering.ID, false)
	if inactive.Active || inactive.PriceCents != offering.PriceCents || inactive.DurationMinutes != offering.DurationMinutes || readProfessionalSecurityState(t, id) != inactiveBefore {
		t.Fatal("inactive configuration changed registrations")
	}
	setProfessionalPair(t, app, id, offering.ID, true)
	inactivePair := readProfessionalPair(t, id, offering.ID)
	assertStatus(t, app.request(t, http.MethodPost, "/api/v1/professionals/"+id+"/reactivation", map[string]string{"temporary_password": "outra-senha"}), http.StatusOK)
	assertStatus(t, app.request(t, http.MethodPut, "/api/v1/services/"+offering.ID+"/status", map[string]bool{"active": true}), http.StatusOK)
	if readProfessionalPair(t, id, offering.ID) != inactivePair {
		t.Fatal("reactivation changed pair")
	}
	var started bool
	if err := integrationPool.QueryRow(context.Background(), `SELECT operational_data_started_at IS NOT NULL FROM establishments WHERE id=(SELECT establishment_id FROM users WHERE id=$1)`, id).Scan(&started); err != nil || started {
		t.Fatalf("operational marker changed: %v %v", started, err)
	}
}

func TestProfessionalServicesMutationRefusesUntrustedUnauthorizedAndForeignTargets(t *testing.T) {
	app := newTestApplication(t)
	id := createManagementProfessional(t, app)
	offering := createCatalogOffering(t, app, "Corte", 100)
	path := "/api/v1/professionals/" + id + "/services/" + offering.ID
	body := map[string]bool{"associated": true}
	for _, test := range []struct {
		origin       string
		csrf, access bool
		status       int
	}{
		{trustedOrigin, false, true, http.StatusForbidden}, {"https://external.example.test", true, true, http.StatusForbidden}, {trustedOrigin, true, false, http.StatusUnauthorized},
	} {
		assertStatus(t, catalogRawRequest(t, app, http.MethodPut, path, body, test.origin, test.csrf, test.access), test.status)
	}
	for _, target := range []string{"/api/v1/professionals/invalid/services/" + offering.ID, "/api/v1/professionals/" + id + "/services/invalid"} {
		assertStatus(t, app.request(t, http.MethodPut, target, body), http.StatusBadRequest)
	}
	var ownerID string
	if err := integrationPool.QueryRow(context.Background(), `SELECT id FROM users WHERE role='owner'`).Scan(&ownerID); err != nil {
		t.Fatal(err)
	}
	for _, associated := range []bool{true, false} {
		for _, target := range []string{"/api/v1/professionals/" + uuid.NewString() + "/services/" + offering.ID, "/api/v1/professionals/" + ownerID + "/services/" + offering.ID, "/api/v1/professionals/" + id + "/services/" + uuid.NewString()} {
			assertStatus(t, app.request(t, http.MethodPut, target, map[string]bool{"associated": associated}), http.StatusNotFound)
		}
	}
	other := defaultOwnerRegistration()
	other.Email = "other@example.test"
	other.Slug = "other-salon"
	registerAndConfirmOwner(t, app, other)
	otherID := createServicesProfessional(t, app, "Bia", "bia@example.test")
	otherOffering := createCatalogOffering(t, app, "Outro", 200)
	for _, associated := range []bool{true, false} {
		for _, target := range []string{path, "/api/v1/professionals/" + otherID + "/services/" + offering.ID, "/api/v1/professionals/" + id + "/services/" + otherOffering.ID} {
			assertStatus(t, app.request(t, http.MethodPut, target, map[string]bool{"associated": associated}), http.StatusNotFound)
		}
	}
	assertStatus(t, app.request(t, http.MethodPost, "/api/v1/auth/login", map[string]string{"email": "ana@example.test", "password": "senha-temporaria"}), http.StatusOK)
	assertStatus(t, app.request(t, http.MethodPost, "/api/v1/auth/password/change", map[string]string{"new_password": "senha-definitiva"}), http.StatusNoContent)
	assertStatus(t, app.request(t, http.MethodPut, path, body), http.StatusForbidden)
	assertStatus(t, app.request(t, http.MethodPost, "/api/v1/auth/login", map[string]string{"email": "owner@example.test", "password": "senha-segura"}), http.StatusOK)
	setProfessionalPair(t, app, id, offering.ID, true)
	before := readProfessionalPair(t, id, offering.ID)
	expireCatalogTrial(t)
	response := app.request(t, http.MethodPut, path, map[string]bool{"associated": false})
	assertStatus(t, response, http.StatusForbidden)
	var failure struct{ Code string }
	decodeJSON(t, response, &failure)
	if failure.Code != "read_only" || readProfessionalPair(t, id, offering.ID) != before {
		t.Fatal("read-only refusal changed pair or code")
	}
}

func TestProfessionalServicesMutationRejectsInvalidBodyWithoutWriting(t *testing.T) {
	app := newTestApplication(t)
	id := createManagementProfessional(t, app)
	offering := createCatalogOffering(t, app, "Corte", 100)
	for _, body := range []string{`{}`, `null`, `{"associated":null}`, `{"associated":1}`, `{"associated":"true"}`, `{"associated":true,"active":true}`, `{"associated":true} {}`, `{"associated":`, `{"associated":true}` + strings.Repeat(" ", 8192)} {
		request, err := http.NewRequestWithContext(context.Background(), http.MethodPut, app.server.URL+"/api/v1/professionals/"+id+"/services/"+offering.ID, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Origin", trustedOrigin)
		request.Header.Set("X-CSRF-Token", app.cookie(t, csrfCookieName).Value)
		response, err := app.client.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		assertStatus(t, response, http.StatusBadRequest)
		response.Body.Close()
	}
	if getProfessionalServicesPage(t, app, "/api/v1/professionals/"+id+"/services").Items[0].Associated {
		t.Fatal("invalid request wrote pair")
	}
}

func TestProfessionalServicesConcurrentWritesPreserveEachPair(t *testing.T) {
	app := newTestApplication(t)
	id := createManagementProfessional(t, app)
	first := createCatalogOffering(t, app, "Corte", 100)
	second := createCatalogOffering(t, app, "Barba", 200)
	run := func(targets []string, states []bool) {
		t.Helper()
		var workers sync.WaitGroup
		start := make(chan struct{})
		failures := make(chan string, len(targets))
		for index, target := range targets {
			workers.Add(1)
			go func(target string, associated bool) {
				defer workers.Done()
				<-start
				response := app.request(t, http.MethodPut, "/api/v1/professionals/"+id+"/services/"+target, map[string]bool{"associated": associated})
				defer response.Body.Close()
				var record professionalOfferingRecord
				if response.StatusCode != http.StatusOK || json.NewDecoder(response.Body).Decode(&record) != nil || record.ID != target || record.Associated != associated {
					failures <- "concurrent operation did not confirm desired state"
				}
			}(target, states[index])
		}
		close(start)
		workers.Wait()
		close(failures)
		for failure := range failures {
			t.Error(failure)
		}
	}
	targets := make([]string, 12)
	states := make([]bool, 12)
	for index := range targets {
		targets[index] = first.ID
		states[index] = true
	}
	run(targets, states)
	initial := readProfessionalPair(t, id, first.ID)
	for index := range states {
		states[index] = index%2 == 0
	}
	run(targets, states)
	if readProfessionalPair(t, id, first.ID).ID != initial.ID {
		t.Fatal("same-pair race replaced identity")
	}
	run([]string{first.ID, second.ID}, []bool{false, true})
	if readProfessionalPair(t, id, first.ID).Active || !readProfessionalPair(t, id, second.ID).Active {
		t.Fatal("different pairs overwrote each other")
	}
	var count int
	if err := integrationPool.QueryRow(context.Background(), `SELECT count(*) FROM professional_services WHERE professional_id=$1`, id).Scan(&count); err != nil || count != 2 {
		t.Fatalf("pairs=%d err=%v", count, err)
	}
}
