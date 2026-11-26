//go:build integration

package integration_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptrace"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	api "barbershop-app/backend/openapi/generated"

	"github.com/jackc/pgx/v5"
)

const (
	initialSchedule = `{"periods":[{"weekday":1,"start_time":"08:00","end_time":"12:00"},{"weekday":2,"start_time":"09:00","end_time":"12:00"}]}`
	firstSchedule   = `{"periods":[{"weekday":3,"start_time":"08:00","end_time":"10:00"},{"weekday":3,"start_time":"10:00","end_time":"12:00"}]}`
	secondSchedule  = `{"periods":[{"weekday":5,"start_time":"13:00","end_time":"15:00"},{"weekday":6,"start_time":"15:00","end_time":"17:00"}]}`
)

type scheduleRequestResult struct {
	response *http.Response
	err      error
}

func startScheduleRequest(client *http.Client, request *http.Request) <-chan scheduleRequestResult {
	results := make(chan scheduleRequestResult, 1)
	go func() {
		response, err := client.Do(request)
		results <- scheduleRequestResult{response, err}
	}()
	return results
}

func finishScheduleRequest(t *testing.T, ctx context.Context, results <-chan scheduleRequestResult) *http.Response {
	t.Helper()
	select {
	case result := <-results:
		if result.err != nil {
			t.Fatal(result.err)
		}
		t.Cleanup(func() { _ = result.response.Body.Close() })
		return result.response
	case <-ctx.Done():
		t.Fatal(ctx.Err())
		return nil
	}
}

func sendScheduleBody(t *testing.T, app *testApplication, id, input string) *http.Response {
	t.Helper()
	request := schedulePutRequest(t, context.Background(), app, id, strings.NewReader(input))
	response, err := app.client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })
	return response
}

func TestWeeklyScheduleWriteRollsBackEntireWeekOnInsertFailure(t *testing.T) {
	app := newTestApplication(t)
	id := createManagementProfessional(t, app)
	initial := decodeSavedSchedule(t, sendScheduleBody(t, app, id, initialSchedule), id)
	ctx := context.Background()
	if _, err := integrationPool.Exec(ctx, `ALTER TABLE professional_work_periods
		ADD CONSTRAINT reject_test_weekday CHECK (weekday <> 6)`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := integrationPool.Exec(context.Background(), `ALTER TABLE professional_work_periods DROP CONSTRAINT reject_test_weekday`); err != nil {
			t.Error(err)
		}
	})
	assertScheduleError(t, sendScheduleBody(t, app, id, secondSchedule), http.StatusInternalServerError, "internal_error")
	if got := readSchedule(t, app, id); !reflect.DeepEqual(got, initial) {
		t.Fatalf("insert failure changed the original week: %+v", got)
	}
}

func blockScheduleTestInsert(t *testing.T, ctx context.Context) pgx.Tx {
	t.Helper()
	if _, err := integrationPool.Exec(ctx, `
		CREATE FUNCTION gate_test_schedule() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN
			PERFORM pg_advisory_xact_lock(874637);
			RETURN NEW;
		END;
		$$;
		CREATE TRIGGER gate_test_schedule_insert BEFORE INSERT ON professional_work_periods
			FOR EACH ROW WHEN (NEW.weekday = 3) EXECUTE FUNCTION gate_test_schedule()
	`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := integrationPool.Exec(context.Background(), `
			DROP TRIGGER gate_test_schedule_insert ON professional_work_periods;
			DROP FUNCTION gate_test_schedule()
		`); err != nil {
			t.Error(err)
		}
	})
	gate, err := integrationPool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = gate.Rollback(context.Background()) })
	if _, err := gate.Exec(ctx, `SELECT pg_advisory_xact_lock(874637)`); err != nil {
		t.Fatal(err)
	}
	return gate
}

func TestWeeklyScheduleWriteSerializesConcurrentReplacementsAndReadsWholeWeek(t *testing.T) {
	app := newTestApplication(t)
	id := createManagementProfessional(t, app)
	initial := decodeSavedSchedule(t, sendScheduleBody(t, app, id, initialSchedule), id)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Pause the first insert after its transaction has executed the DELETE.
	// Readers must still see the old week and the second writer must wait.
	gate := blockScheduleTestInsert(t, ctx)

	first := startScheduleRequest(app.client, schedulePutRequest(t, ctx, app, id, strings.NewReader(firstSchedule)))
	waitForBlockedDatabaseQuery(t, ctx, "InsertProfessionalWorkPeriods")
	if got := readSchedule(t, app, id); !reflect.DeepEqual(got, initial) {
		t.Fatalf("reader saw uncommitted or partial week: %+v", got)
	}

	second := startScheduleRequest(app.client, schedulePutRequest(t, ctx, app, id, strings.NewReader(secondSchedule)))
	waitForBlockedDatabaseQuery(t, ctx, "LockProfessionalUser")
	if err := gate.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	firstWeek := decodeSavedSchedule(t, finishScheduleRequest(t, ctx, first), id)
	secondWeek := decodeSavedSchedule(t, finishScheduleRequest(t, ctx, second), id)
	if !reflect.DeepEqual(firstWeek.Periods, []api.ProfessionalWorkPeriod{
		{Weekday: 3, StartTime: "08:00", EndTime: "10:00"},
		{Weekday: 3, StartTime: "10:00", EndTime: "12:00"},
	}) {
		t.Fatalf("first confirmation was not its complete week: %+v", firstWeek)
	}
	if !reflect.DeepEqual(secondWeek.Periods, []api.ProfessionalWorkPeriod{
		{Weekday: 5, StartTime: "13:00", EndTime: "15:00"},
		{Weekday: 6, StartTime: "15:00", EndTime: "17:00"},
	}) {
		t.Fatalf("second confirmation was not its complete week: %+v", secondWeek)
	}
	if got := readSchedule(t, app, id); !reflect.DeepEqual(got, secondWeek) {
		t.Fatalf("read after concurrent writes = %+v, want last accepted week %+v", got, secondWeek)
	}
}

func TestWeeklyScheduleWriteConfirmsTimeZoneAfterConcurrentSettingsUpdate(t *testing.T) {
	app := newTestApplication(t)
	id := createManagementProfessional(t, app)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	gate := blockScheduleTestInsert(t, ctx)

	pending := startScheduleRequest(app.client, schedulePutRequest(t, ctx, app, id, strings.NewReader(firstSchedule)))
	waitForBlockedDatabaseQuery(t, ctx, "InsertProfessionalWorkPeriods")
	const updatedZone = "America/Manaus"
	assertStatus(t, app.request(t, http.MethodPatch, "/api/v1/establishment/settings", map[string]string{"time_zone": updatedZone}), http.StatusNoContent)
	if err := gate.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	confirmed := decodeSavedSchedule(t, finishScheduleRequest(t, ctx, pending), id)
	if got := readSchedule(t, app, id); confirmed.TimeZone != updatedZone || !reflect.DeepEqual(got, confirmed) {
		t.Fatalf("schedule confirmation = %+v, read = %+v, updated time zone = %s", confirmed, got, updatedZone)
	}
}

func TestWeeklyScheduleWriteRechecksTrialAfterWaitingForProfessionalLock(t *testing.T) {
	app := newTestApplication(t)
	id := createManagementProfessional(t, app)
	initial := decodeSavedSchedule(t, sendScheduleBody(t, app, id, initialSchedule), id)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	gate, err := integrationPool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer gate.Rollback(context.Background())
	if _, err := gate.Exec(ctx, `SELECT id FROM users WHERE id = $1 FOR UPDATE`, id); err != nil {
		t.Fatal(err)
	}

	pending := startScheduleRequest(app.client, schedulePutRequest(t, ctx, app, id, strings.NewReader(secondSchedule)))
	waitForBlockedDatabaseQuery(t, ctx, "LockProfessionalUser")
	// Simulate expiration only after the request has passed the initial access check.
	expireCatalogTrial(t)
	if err := gate.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	assertScheduleError(t, finishScheduleRequest(t, ctx, pending), http.StatusForbidden, "read_only")
	if got := readSchedule(t, app, id); got.CanWrite || !reflect.DeepEqual(got.Periods, initial.Periods) {
		t.Fatalf("expired trial did not preserve the original week: %+v", got)
	}
}

func TestWeeklyScheduleWriteRejectsSelfEditAuthenticatedBeforeReactivation(t *testing.T) {
	app := newTestApplication(t)
	id := createManagementProfessional(t, app)
	initial := decodeSavedSchedule(t, sendScheduleBody(t, app, id, initialSchedule), id)
	peer := independentClient(t, app)
	loginProfessional(t, peer)
	assertStatus(t, peer.request(t, http.MethodPost, "/api/v1/auth/password/change", map[string]string{"new_password": "senha-definitiva"}), http.StatusNoContent)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	request := schedulePutRequest(t, ctx, peer, id, reader)
	request.ContentLength = int64(len(firstSchedule))
	request.Header.Set("Expect", "100-continue")
	request.AddCookie(peer.cookie(t, csrfCookieName))
	request.AddCookie(peer.cookie(t, accessCookieName))
	bodyRequested := make(chan struct{})
	var once sync.Once
	request = request.WithContext(httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
		Got100Continue: func() { once.Do(func() { close(bodyRequested) }) },
	}))
	results := startScheduleRequest(&http.Client{Transport: peer.client.Transport}, request)
	// net/http sends 100 Continue when the handler first reads the body,
	// after middleware has established the old authenticated principal.
	select {
	case <-bodyRequested:
	case result := <-results:
		if result.response != nil {
			_ = result.response.Body.Close()
		}
		t.Fatalf("schedule handler did not reach the body read: %v", result.err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	deactivateManagementProfessional(t, app, id)
	assertStatus(t, app.request(t, http.MethodPost, "/api/v1/professionals/"+id+"/reactivation", map[string]string{"temporary_password": "nova-temporaria"}), http.StatusOK)
	if _, err := io.WriteString(writer, firstSchedule); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	assertScheduleError(t, finishScheduleRequest(t, ctx, results), http.StatusUnauthorized, "authentication_required")
	if got := readSchedule(t, app, id); !reflect.DeepEqual(got, initial) {
		t.Fatalf("stale authenticated request changed the schedule: %+v", got)
	}
}
