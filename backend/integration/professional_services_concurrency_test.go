//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	db "barbershop-app/backend/db/generated"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestProfessionalServicesTransitionDateFollowsLockOrder(t *testing.T) {
	app := newTestApplication(t)
	professionalID := createManagementProfessional(t, app)
	offering := createCatalogOffering(t, app, "Corte", 100)
	setProfessionalPair(t, app, professionalID, offering.ID, true)
	original := readProfessionalPair(t, professionalID, offering.ID)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	transaction, err := integrationPool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer transaction.Rollback(context.Background())
	var establishmentID pgtype.UUID
	if err := transaction.QueryRow(ctx, `SELECT establishment_id FROM professional_services WHERE id=$1 FOR UPDATE`, original.ID).Scan(&establishmentID); err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPut, app.server.URL+"/api/v1/professionals/"+professionalID+"/services/"+offering.ID, strings.NewReader(`{"associated":true}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", trustedOrigin)
	request.Header.Set("X-CSRF-Token", app.cookie(t, csrfCookieName).Value)
	responses := make(chan *http.Response, 1)
	failures := make(chan error, 1)
	go func() {
		response, err := app.client.Do(request)
		if err != nil {
			failures <- err
			return
		}
		responses <- response
	}()
	waitForBlockedDatabaseQuery(t, ctx, "SetProfessionalOffering")
	// Start the removal after the waiting PUT's statement has already begun.
	// The next transition must use lock order rather than statement start order.
	parameters := db.SetProfessionalOfferingParams{
		EstablishmentID: establishmentID,
		ProfessionalID:  pgtype.UUID{Bytes: uuid.MustParse(professionalID), Valid: true},
		ServiceID:       pgtype.UUID{Bytes: uuid.MustParse(offering.ID), Valid: true},
		Associated:      false,
	}
	if active, err := db.New(transaction).SetProfessionalOffering(ctx, parameters); err != nil || active {
		t.Fatalf("remove association: active=%v err=%v", active, err)
	}
	var removedAt time.Time
	if err := transaction.QueryRow(ctx, `SELECT updated_at FROM professional_services WHERE id=$1`, original.ID).Scan(&removedAt); err != nil {
		t.Fatal(err)
	}
	if err := transaction.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case response := <-responses:
		assertStatus(t, response, http.StatusOK)
		var record professionalOfferingRecord
		decodeJSON(t, response, &record)
		if record.ID != offering.ID || !record.Associated {
			t.Fatalf("wrong confirmation: %+v", record)
		}
	case err := <-failures:
		t.Fatal(err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	restored := readProfessionalPair(t, professionalID, offering.ID)
	if restored.ID != original.ID || restored.CreatedAt != original.CreatedAt || !restored.Active || !restored.UpdatedAt.After(removedAt) {
		t.Fatalf("transition did not follow lock order: removed_at=%v restored=%+v", removedAt, restored)
	}
}
