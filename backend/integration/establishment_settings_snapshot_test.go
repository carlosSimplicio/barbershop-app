//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestEstablishmentSettingsReadKeepsProfileAndLinksInOneSnapshot(t *testing.T) {
	application := newTestApplication(t)
	registration := defaultOwnerRegistration()
	registerAndConfirmOwner(t, application, registration)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tx, err := integrationPool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// Block link reads while a profile update is prepared in the same transaction.
	if _, err := tx.Exec(ctx, "LOCK TABLE establishment_social_links IN ACCESS EXCLUSIVE MODE"); err != nil {
		t.Fatal(err)
	}
	responses := make(chan *http.Response, 1)
	go func() { responses <- application.request(t, http.MethodGet, "/api/v1/establishment/settings", nil) }()
	waitForBlockedDatabaseQuery(t, ctx, "Establishment")
	if _, err := tx.Exec(ctx, "UPDATE establishments SET name='Updated profile' WHERE slug=$1", registration.Slug); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, "INSERT INTO establishment_social_links (establishment_id, position, label, url) SELECT id, 0, 'Updated link', 'https://example.test/updated' FROM establishments WHERE slug=$1", registration.Slug); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	var response *http.Response
	select {
	case response = <-responses:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	assertStatus(t, response, http.StatusOK)
	var settings establishmentSettingsResponse
	decodeJSON(t, response, &settings)
	oldSnapshot := settings.Name == registration.EstablishmentName && len(settings.SocialLinks) == 0
	newSnapshot := settings.Name == "Updated profile" && len(settings.SocialLinks) == 1 && settings.SocialLinks[0].Label == "Updated link"
	if !oldSnapshot && !newSnapshot {
		t.Fatal("response combined profile and links from different snapshots")
	}
	// A subsequent read must see the complete committed update.
	settings = getEstablishmentSettings(t, application)
	if settings.Name != "Updated profile" || len(settings.SocialLinks) != 1 {
		t.Fatal("committed settings update was not visible")
	}
}
