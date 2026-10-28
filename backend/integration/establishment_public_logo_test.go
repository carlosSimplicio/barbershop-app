//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strconv"
	"testing"
	"time"

	"image/color"
)

func TestGetPublicEstablishmentLogoServesCurrentLogoWithoutAuthentication(t *testing.T) {
	application := newTestApplication(t)
	registerAndConfirmOwner(t, application, defaultOwnerRegistration())
	logo := testLogoPNG(t, color.NRGBA{R: 24, G: 80, B: 160, A: 255})
	assertStatus(t, uploadLogo(t, application, "logo.png", "image/png", logo, ""), http.StatusNoContent)

	response := getPublicLogo(t, application, defaultOwnerRegistration().Slug, "")
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("public logo status = %d, body = %s; want 200", response.StatusCode, body)
	}
	got, err := io.ReadAll(response.Body)
	if err != nil || !bytes.Equal(got, logo) {
		t.Fatalf("public logo bytes match upload = %v, read error = %v", bytes.Equal(got, logo), err)
	}
	firstETag := response.Header.Get("ETag")
	contentHash := sha256.Sum256(logo)
	wantETag := `"` + hex.EncodeToString(contentHash[:]) + `"`
	if firstETag != wantETag || response.Header.Get("Content-Type") != "image/png" || response.Header.Get("Content-Length") != strconv.Itoa(len(logo)) {
		t.Fatalf("initial logo headers: ETag=%q Content-Type=%q Content-Length=%q", firstETag, response.Header.Get("Content-Type"), response.Header.Get("Content-Length"))
	}
	if response.Header.Get("X-Content-Type-Options") != "nosniff" || response.Header.Get("Cache-Control") != "public, no-cache" {
		t.Fatalf("logo security/cache headers: nosniff=%q cache-control=%q", response.Header.Get("X-Content-Type-Options"), response.Header.Get("Cache-Control"))
	}
	conditional := getPublicLogo(t, application, defaultOwnerRegistration().Slug, firstETag)
	if conditional.StatusCode != http.StatusNotModified || conditional.Header.Get("ETag") != firstETag {
		conditional.Body.Close()
		t.Fatalf("matching If-None-Match status=%d ETag=%q", conditional.StatusCode, conditional.Header.Get("ETag"))
	}
	conditionalBody, err := io.ReadAll(conditional.Body)
	conditional.Body.Close()
	if err != nil || len(conditionalBody) != 0 {
		t.Fatalf("304 response body length=%d, err=%v; want empty", len(conditionalBody), err)
	}

	replacement := testLogoPNG(t, color.NRGBA{R: 220, G: 100, B: 30, A: 255})
	assertStatus(t, uploadLogo(t, application, "replacement.png", "image/png", replacement, ""), http.StatusNoContent)
	changed := getPublicLogo(t, application, defaultOwnerRegistration().Slug, firstETag)
	defer changed.Body.Close()
	changedBody, err := io.ReadAll(changed.Body)
	if changed.StatusCode != http.StatusOK || err != nil || !bytes.Equal(changedBody, replacement) || changed.Header.Get("ETag") == firstETag {
		t.Fatalf("replacement response status=%d bodyMatches=%v oldETag=%q newETag=%q err=%v", changed.StatusCode, bytes.Equal(changedBody, replacement), firstETag, changed.Header.Get("ETag"), err)
	}

	establishmentID := authenticatedEstablishmentID(t, application)
	if _, err := integrationPool.Exec(context.Background(), `UPDATE establishment_trials SET starts_at = $1, ends_at = $2 WHERE establishment_id = $3`, time.Now().Add(-15*24*time.Hour), time.Now().Add(-24*time.Hour), establishmentID); err != nil {
		t.Fatalf("expire trial fixture: %v", err)
	}
	expiredTrial := getPublicLogo(t, application, defaultOwnerRegistration().Slug, "")
	defer expiredTrial.Body.Close()
	if expiredTrial.StatusCode != http.StatusOK {
		t.Fatalf("expired-trial public logo status = %d, want 200", expiredTrial.StatusCode)
	}
}

func TestGetPublicEstablishmentLogoReturnsNotFoundForUnavailableProfiles(t *testing.T) {
	application := newTestApplication(t)
	registerAndConfirmOwner(t, application, defaultOwnerRegistration())
	for _, slug := range []string{"missing-establishment", defaultOwnerRegistration().Slug} {
		response := getPublicLogo(t, application, slug, "")
		assertStatus(t, response, http.StatusNotFound)
	}

	logo := testLogoPNG(t, color.NRGBA{R: 10, G: 20, B: 30, A: 255})
	assertStatus(t, uploadLogo(t, application, "logo.png", "image/png", logo, ""), http.StatusNoContent)
	if _, err := integrationPool.Exec(context.Background(), `UPDATE establishments SET slug = 'api' WHERE id = $1`, authenticatedEstablishmentID(t, application)); err != nil {
		t.Fatalf("create reserved legacy slug fixture: %v", err)
	}
	response := getPublicLogo(t, application, "api", "")
	assertStatus(t, response, http.StatusNotFound)
}

func TestGetPublicEstablishmentLogoResolvesTenantAndReportsStorageFailureSafely(t *testing.T) {
	application := newTestApplication(t)
	firstOwner := defaultOwnerRegistration()
	registerAndConfirmOwner(t, application, firstOwner)
	firstID := authenticatedEstablishmentID(t, application)
	firstImage := testLogoPNG(t, color.NRGBA{R: 30, G: 60, B: 90, A: 255})
	assertStatus(t, uploadLogo(t, application, "logo.png", "image/png", firstImage, ""), http.StatusNoContent)
	secondOwner := ownerRegistration{EstablishmentName: "Outro Salão", Slug: "outro-salao", TimeZone: "America/Sao_Paulo", Email: "other@example.test", Password: "senha-segura"}
	registerAndConfirmOwner(t, application, secondOwner)
	secondID := authenticatedEstablishmentID(t, application)
	secondImage := testLogoPNG(t, color.NRGBA{R: 110, G: 140, B: 170, A: 255})
	assertStatus(t, uploadLogo(t, application, "logo.png", "image/png", secondImage, ""), http.StatusNoContent)

	firstResponse := getPublicLogo(t, application, firstOwner.Slug, "")
	firstBody, err := io.ReadAll(firstResponse.Body)
	firstResponse.Body.Close()
	if err != nil || firstResponse.StatusCode != http.StatusOK || !bytes.Equal(firstBody, firstImage) {
		t.Fatalf("first tenant logo status=%d contentMatches=%v err=%v", firstResponse.StatusCode, bytes.Equal(firstBody, firstImage), err)
	}
	secondResponse := getPublicLogo(t, application, secondOwner.Slug, "")
	secondBody, err := io.ReadAll(secondResponse.Body)
	secondResponse.Body.Close()
	if err != nil || secondResponse.StatusCode != http.StatusOK || !bytes.Equal(secondBody, secondImage) {
		t.Fatalf("second tenant logo status=%d contentMatches=%v err=%v", secondResponse.StatusCode, bytes.Equal(secondBody, secondImage), err)
	}

	key, _ := currentLogoContents(t, application, secondID)
	if err := application.logoStorage.Delete(context.Background(), key); err != nil {
		t.Fatalf("remove object to simulate unavailable storage: %v", err)
	}
	failedStorageResponse := getPublicLogo(t, application, secondOwner.Slug, "")
	defer failedStorageResponse.Body.Close()
	assertStatus(t, failedStorageResponse, http.StatusInternalServerError)
	body, err := io.ReadAll(failedStorageResponse.Body)
	if err != nil || bytes.Contains(body, []byte(string(key))) || bytes.Contains(body, []byte("LOGO_STORAGE_DIR")) {
		t.Fatalf("storage failure response leaks internal data or failed to read: %q, err=%v", body, err)
	}
	if firstID == secondID {
		t.Fatal("test owners unexpectedly share a tenant")
	}
}

func getPublicLogo(t *testing.T, application *testApplication, slug, ifNoneMatch string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, application.server.URL+"/api/v1/public/establishments/"+slug+"/logo", nil)
	if err != nil {
		t.Fatal(err)
	}
	if ifNoneMatch != "" {
		request.Header.Set("If-None-Match", ifNoneMatch)
	}
	response, err := application.server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	return response
}
