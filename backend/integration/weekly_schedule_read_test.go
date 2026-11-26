//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	api "barbershop-app/backend/openapi/generated"

	"github.com/google/uuid"
)

func schedulePath(professionalID string) string {
	return "/api/v1/professionals/" + professionalID + "/weekly-schedule"
}

func readSchedule(t *testing.T, app *testApplication, professionalID string) api.ProfessionalWeeklyScheduleResponse {
	t.Helper()
	response := app.request(t, http.MethodGet, schedulePath(professionalID), nil)
	assertStatus(t, response, http.StatusOK)
	var week api.ProfessionalWeeklyScheduleResponse
	decodeJSON(t, response, &week)
	if week.ProfessionalId.String() != professionalID || week.Periods == nil {
		t.Fatalf("invalid schedule response: %+v", week)
	}
	return week
}

func TestWeeklyScheduleReadReturnsLocalPeriodsAndCurrentRights(t *testing.T) {
	app := newTestApplication(t)
	id := createManagementProfessional(t, app)
	week := readSchedule(t, app, id)
	if week.TimeZone != "America/Sao_Paulo" || !week.CanWrite || len(week.Periods) != 0 {
		t.Fatalf("initial schedule = %+v", week)
	}
	ctx := context.Background()
	if _, err := integrationPool.Exec(ctx, `INSERT INTO professional_work_periods
		(establishment_id, professional_id, weekday, start_minute, end_minute)
		SELECT p.establishment_id, p.user_id, input.weekday, input.start_minute, input.end_minute
		FROM professionals AS p CROSS JOIN (VALUES (7, 1383, 1440), (1, 480, 720), (1, 0, 10))
			AS input(weekday, start_minute, end_minute)
		WHERE p.user_id = $1`, id); err != nil {
		t.Fatal(err)
	}
	want := []api.ProfessionalWorkPeriod{
		{Weekday: 1, StartTime: "00:00", EndTime: "00:10"},
		{Weekday: 1, StartTime: "08:00", EndTime: "12:00"},
		{Weekday: 7, StartTime: "23:03", EndTime: "24:00"},
	}
	professional := independentClient(t, app)
	loginProfessional(t, professional)
	response := professional.request(t, http.MethodGet, schedulePath(id), nil)
	assertStatus(t, response, http.StatusForbidden)
	var blocked api.ErrorResponse
	decodeJSON(t, response, &blocked)
	if blocked.Code != "password_change_required" {
		t.Fatalf("pending password change = %q", blocked.Code)
	}
	assertStatus(t, professional.request(t, http.MethodPost, "/api/v1/auth/password/change", map[string]string{"new_password": "senha-definitiva"}), http.StatusNoContent)

	for _, client := range []*testApplication{app, professional} {
		week = readSchedule(t, client, id)
		if !week.CanWrite || !reflect.DeepEqual(week.Periods, want) {
			t.Fatalf("active week = %+v", week)
		}
	}
	expireCatalogTrial(t)
	for _, client := range []*testApplication{app, professional} {
		week = readSchedule(t, client, id)
		if week.CanWrite || !reflect.DeepEqual(week.Periods, want) {
			t.Fatalf("expired week = %+v", week)
		}
	}
	if _, err := integrationPool.Exec(ctx, `UPDATE users SET active = false WHERE id = $1`, id); err != nil {
		t.Fatal(err)
	}
	week = readSchedule(t, app, id)
	if week.CanWrite || !reflect.DeepEqual(week.Periods, want) {
		t.Fatalf("owner could not read inactive professional: %+v", week)
	}
}

func TestWeeklyScheduleReadScopesTargetToSession(t *testing.T) {
	app := newTestApplication(t)
	missing := uuid.NewString()
	assertStatus(t, app.request(t, http.MethodGet, schedulePath(missing), nil), http.StatusUnauthorized)
	id := createManagementProfessional(t, app)
	colleague := createServicesProfessional(t, app, "Bia", "bia@example.test")
	var ownerID string
	if err := integrationPool.QueryRow(context.Background(), `SELECT id FROM users WHERE role = 'owner'`).Scan(&ownerID); err != nil {
		t.Fatal(err)
	}
	assertStatus(t, app.request(t, http.MethodGet, schedulePath("invalid"), nil), http.StatusBadRequest)
	for _, target := range []string{missing, ownerID} {
		assertStatus(t, app.request(t, http.MethodGet, schedulePath(target), nil), http.StatusNotFound)
	}

	other := independentClient(t, app)
	otherOwner := defaultOwnerRegistration()
	otherOwner.Email = "other@example.test"
	otherOwner.Slug = "outro-salao"
	registerAndConfirmOwner(t, other, otherOwner)
	otherID := createServicesProfessional(t, other, "Caio", "caio@example.test")
	assertStatus(t, app.request(t, http.MethodGet, schedulePath(otherID), nil), http.StatusNotFound)
	assertStatus(t, other.request(t, http.MethodGet, schedulePath(id), nil), http.StatusNotFound)

	professional := independentClient(t, app)
	loginProfessional(t, professional)
	assertStatus(t, professional.request(t, http.MethodPost, "/api/v1/auth/password/change", map[string]string{"new_password": "senha-definitiva"}), http.StatusNoContent)
	for _, target := range []string{colleague, otherID, ownerID, missing} {
		response := professional.request(t, http.MethodGet, schedulePath(target), nil)
		assertStatus(t, response, http.StatusNotFound)
		var failure api.ErrorResponse
		decodeJSON(t, response, &failure)
		if failure.Code != "not_found" {
			t.Fatalf("unexpected refusal code %q", failure.Code)
		}
	}
}
