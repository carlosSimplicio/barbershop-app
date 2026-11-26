//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"barbershop-app/backend/internal/professionals/schedule"
	api "barbershop-app/backend/openapi/generated"

	"github.com/google/uuid"
)

func saveSchedule(t *testing.T, app *testApplication, id string, periods []api.ProfessionalWorkPeriod) api.ProfessionalWeeklyScheduleResponse {
	t.Helper()
	return decodeSavedSchedule(t, app.request(t, http.MethodPut, schedulePath(id), api.ProfessionalWeeklyScheduleRequest{Periods: periods}), id)
}

func decodeSavedSchedule(t *testing.T, response *http.Response, id string) api.ProfessionalWeeklyScheduleResponse {
	t.Helper()
	assertStatus(t, response, http.StatusOK)
	var week api.ProfessionalWeeklyScheduleResponse
	decodeJSON(t, response, &week)
	if week.ProfessionalId.String() != id || !week.CanWrite || week.Periods == nil {
		t.Fatalf("invalid saved schedule response: %+v", week)
	}
	return week
}

func assertScheduleError(t *testing.T, response *http.Response, status int, code string) {
	t.Helper()
	assertStatus(t, response, status)
	var failure api.ErrorResponse
	decodeJSON(t, response, &failure)
	if failure.Code != code || failure.RequestId == "" {
		t.Fatalf("failure = %+v, want %s and a request ID", failure, code)
	}
}

func schedulePutRequest(t *testing.T, ctx context.Context, app *testApplication, id string, body io.Reader) *http.Request {
	t.Helper()
	request, err := http.NewRequestWithContext(ctx, http.MethodPut, app.server.URL+schedulePath(id), body)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", trustedOrigin)
	request.Header.Set("X-CSRF-Token", app.cookie(t, csrfCookieName).Value)
	return request
}

func TestWeeklyScheduleWriteChecksOwnerSelfTenantTrialAndCSRF(t *testing.T) {
	app := newTestApplication(t)
	id := createManagementProfessional(t, app)
	colleague := createServicesProfessional(t, app, "Bia", "bia@example.test")
	periods := []api.ProfessionalWorkPeriod{{Weekday: 1, StartTime: "08:00", EndTime: "12:00"}}
	week := api.ProfessionalWeeklyScheduleRequest{Periods: periods}
	path := schedulePath(id)
	assertScheduleError(t, catalogRawRequest(t, app, http.MethodPut, path, week, trustedOrigin, false, true), http.StatusForbidden, "csrf_failed")
	assertScheduleError(t, catalogRawRequest(t, app, http.MethodPut, path, week, "https://other.example.test", true, true), http.StatusForbidden, "csrf_failed")
	assertScheduleError(t, catalogRawRequest(t, app, http.MethodPut, path, week, trustedOrigin, true, false), http.StatusUnauthorized, "authentication_required")

	professional := independentClient(t, app)
	loginProfessional(t, professional)
	assertScheduleError(t, professional.request(t, http.MethodPut, path, week), http.StatusForbidden, "password_change_required")
	assertStatus(t, professional.request(t, http.MethodPost, "/api/v1/auth/password/change", map[string]string{"new_password": "senha-definitiva"}), http.StatusNoContent)
	assertScheduleError(t, professional.request(t, http.MethodPut, schedulePath(colleague), week), http.StatusNotFound, "not_found")
	if len(readSchedule(t, app, id).Periods) != 0 {
		t.Fatal("refused requests changed the user's schedule")
	}
	if got := saveSchedule(t, professional, id, periods); !reflect.DeepEqual(got.Periods, periods) {
		t.Fatalf("self edit = %+v", got)
	}

	other := independentClient(t, app)
	otherOwner := defaultOwnerRegistration()
	otherOwner.Email = "other@example.test"
	otherOwner.Slug = "outra-barbearia"
	registerAndConfirmOwner(t, other, otherOwner)
	assertScheduleError(t, other.request(t, http.MethodPut, path, week), http.StatusNotFound, "not_found")
	assertScheduleError(t, app.request(t, http.MethodPut, schedulePath(uuid.NewString()), week), http.StatusNotFound, "not_found")
	deactivateManagementProfessional(t, app, id)
	periods[0].EndTime = "13:00"
	if got := saveSchedule(t, app, id, periods); !reflect.DeepEqual(got.Periods, periods) {
		t.Fatalf("owner edit of inactive professional = %+v", got)
	}
	expireCatalogTrial(t)
	assertScheduleError(t, app.request(t, http.MethodPut, path, api.ProfessionalWeeklyScheduleRequest{Periods: []api.ProfessionalWorkPeriod{}}), http.StatusForbidden, "read_only")
	assertScheduleError(t, app.request(t, http.MethodPut, schedulePath(uuid.NewString()), week), http.StatusNotFound, "not_found")
	if got := readSchedule(t, app, id); got.CanWrite || !reflect.DeepEqual(got.Periods, periods) {
		t.Fatalf("read-only write changed the week: %+v", got)
	}
}

func TestWeeklyScheduleWriteRejectsInvalidWeeksAndPreservesTimeZonePolicy(t *testing.T) {
	app := newTestApplication(t)
	id := createManagementProfessional(t, app)
	first := api.ProfessionalWorkPeriod{Weekday: 1, StartTime: "22:00", EndTime: "24:00"}
	second := api.ProfessionalWorkPeriod{Weekday: 2, StartTime: "00:00", EndTime: "02:00"}
	sorted := []api.ProfessionalWorkPeriod{first, second}
	input := []api.ProfessionalWorkPeriod{second, first}
	for range 2 {
		got := saveSchedule(t, app, id, input)
		if got.TimeZone != "America/Sao_Paulo" || !reflect.DeepEqual(got.Periods, sorted) {
			t.Fatalf("saved week = %+v", got)
		}
	}
	for _, body := range []string{
		`{}`, `null`, `{"periods":null}`, `{"periods":{}}`, `{"periods":[null]}`,
		`{"Periods":[]}`, `{"periods":[],"extra":true}`,
		`{"periods":null,"periods":[]}`,
		`{"periods":[{"weekday":null,"weekday":1,"start_time":"08:00","end_time":"12:00"}]}`,
		`{"periods":[{"weekday":null,"start_time":"08:00","end_time":"12:00"}]}`,
		`{"periods":[{"weekday":1,"start_time":null,"end_time":"12:00"}]}`,
		`{"periods":[{"weekday":1,"start_time":"08:00","end_time":null}]}`,
		`{"periods":[{"weekday":1,"Start_Time":"08:00","end_time":"12:00"}]}`,
		`{"periods":[{"weekday":1,"start_time":"08:00","end_time":"12:00","extra":true}]}`,
		`{"periods":[]} {}`,
	} {
		t.Run(body, func(t *testing.T) {
			request := schedulePutRequest(t, context.Background(), app, id, strings.NewReader(body))
			response, err := app.client.Do(request)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			assertScheduleError(t, response, http.StatusBadRequest, "invalid_request")
		})
	}
	for _, invalid := range []api.ProfessionalWorkPeriod{
		{Weekday: 1, StartTime: "23:00", EndTime: "24:00"},
		{Weekday: 1, StartTime: "23:00", EndTime: "01:00"},
	} {
		response := app.request(t, http.MethodPut, schedulePath(id), api.ProfessionalWeeklyScheduleRequest{Periods: []api.ProfessionalWorkPeriod{first, invalid}})
		assertScheduleError(t, response, http.StatusBadRequest, "validation_failed")
	}
	if got := readSchedule(t, app, id); !reflect.DeepEqual(got.Periods, sorted) {
		t.Fatalf("invalid write partially replaced the week: %+v", got)
	}
	assertStatus(t, app.request(t, http.MethodPatch, "/api/v1/establishment/settings", map[string]string{"time_zone": "America/Manaus"}), http.StatusNoContent)
	if got := readSchedule(t, app, id); got.TimeZone != "America/Manaus" || !reflect.DeepEqual(got.Periods, sorted) {
		t.Fatalf("time zone change altered local periods: %+v", got)
	}
	if got := saveSchedule(t, app, id, []api.ProfessionalWorkPeriod{}); got.TimeZone != "America/Manaus" || len(got.Periods) != 0 {
		t.Fatalf("cleared week = %+v", got)
	}
	if got := readSchedule(t, app, id); len(got.Periods) != 0 {
		t.Fatalf("clearing was not persisted: %+v", got)
	}
	var started bool
	if err := integrationPool.QueryRow(context.Background(), `SELECT e.operational_data_started_at IS NOT NULL
		FROM establishments AS e JOIN professionals AS p ON p.establishment_id = e.id WHERE p.user_id = $1`, id).Scan(&started); err != nil || started {
		t.Fatalf("operational data marker present = %v, error = %v", started, err)
	}
}

func TestWeeklyScheduleWriteConfirmsIdenticalWeekWithoutRewritingPeriods(t *testing.T) {
	app := newTestApplication(t)
	id := createManagementProfessional(t, app)
	periods := []api.ProfessionalWorkPeriod{
		{Weekday: 1, StartTime: "08:00", EndTime: "12:00"},
		{Weekday: 1, StartTime: "13:00", EndTime: "18:00"},
	}
	initial := saveSchedule(t, app, id, periods)
	if _, err := integrationPool.Exec(context.Background(), `
		CREATE FUNCTION reject_test_schedule_rewrite() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN
			RAISE EXCEPTION 'the same week must not issue a write';
		END;
		$$;
		CREATE TRIGGER reject_test_schedule_rewrite BEFORE INSERT OR DELETE OR UPDATE ON professional_work_periods
		FOR EACH STATEMENT EXECUTE FUNCTION reject_test_schedule_rewrite()
	`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := integrationPool.Exec(context.Background(), `
			DROP TRIGGER reject_test_schedule_rewrite ON professional_work_periods;
			DROP FUNCTION reject_test_schedule_rewrite()
		`); err != nil {
			t.Error(err)
		}
	})
	if got := saveSchedule(t, app, id, []api.ProfessionalWorkPeriod{periods[1], periods[0]}); !reflect.DeepEqual(got, initial) {
		t.Fatalf("repeated write = %+v, want %+v", got, initial)
	}
}

func TestWeeklyScheduleWriteAcceptsEveryMinuteAndBoundsRequestBytes(t *testing.T) {
	app := newTestApplication(t)
	id := createManagementProfessional(t, app)
	periods := make([]api.ProfessionalWorkPeriod, 0, 10080)
	for day := 1; day <= 7; day++ {
		for minute := 0; minute < 1440; minute++ {
			end := minute + 1
			periods = append(periods, api.ProfessionalWorkPeriod{
				Weekday:   day,
				StartTime: fmt.Sprintf("%02d:%02d", minute/60, minute%60),
				EndTime:   fmt.Sprintf("%02d:%02d", end/60, end%60),
			})
		}
	}
	if got := saveSchedule(t, app, id, periods); !reflect.DeepEqual(got.Periods, periods) {
		t.Fatalf("large week confirmation returned %d of %d expected periods", len(got.Periods), len(periods))
	}
	const emptyWeek = `{"periods":[]}`
	for _, test := range []struct {
		name    string
		chunked bool
	}{
		{"content length", false},
		{"chunked transfer", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			payload := emptyWeek + strings.Repeat(" ", schedule.MaxRequestBytes-len(emptyWeek)+1)
			var body io.Reader = strings.NewReader(payload)
			if test.chunked {
				body = io.LimitReader(body, int64(len(payload)))
			}
			request := schedulePutRequest(t, context.Background(), app, id, body)
			response, err := app.client.Do(request)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			assertScheduleError(t, response, http.StatusRequestEntityTooLarge, "request_too_large")
		})
	}
	if got := readSchedule(t, app, id); !reflect.DeepEqual(got.Periods, periods) {
		t.Fatalf("persisted week after oversized requests returned %d of %d periods", len(got.Periods), len(periods))
	}
}
