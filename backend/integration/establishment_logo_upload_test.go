//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"sync"
	"testing"
	"time"

	db "barbershop-app/backend/db/generated"
	"barbershop-app/backend/internal/establishment"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestPutEstablishmentLogoUploadsAndReplacesValidatedImage(t *testing.T) {
	application := newTestApplication(t)
	registerAndConfirmOwner(t, application, defaultOwnerRegistration())
	establishmentID := authenticatedEstablishmentID(t, application)

	firstImage := testLogoPNG(t, color.NRGBA{R: 12, G: 34, B: 56, A: 255})
	response := uploadLogo(t, application, "logo.png", "image/png", firstImage, "")
	assertStatus(t, response, http.StatusNoContent)
	settings := getEstablishmentSettings(t, application)
	if !settings.IsComplete || !settings.IsPublishable || settings.Logo == nil || settings.Logo.ContentType != "image/png" {
		t.Fatalf("settings after upload = %+v", settings)
	}
	if settings.Logo.URL == nil || *settings.Logo.URL != "/api/v1/public/establishments/"+defaultOwnerRegistration().Slug+"/logo" {
		t.Fatalf("published logo URL = %v", settings.Logo.URL)
	}
	firstKey, firstContents := currentLogoContents(t, application, establishmentID)
	if !bytes.Equal(firstContents, firstImage) {
		t.Fatal("stored logo content differs from uploaded image")
	}

	secondImage := testLogoPNG(t, color.NRGBA{R: 210, G: 80, B: 32, A: 255})
	response = uploadLogo(t, application, "replacement.png", "text/plain", secondImage, "")
	assertStatus(t, response, http.StatusNoContent)
	secondKey, secondContents := currentLogoContents(t, application, establishmentID)
	if secondKey == firstKey || !bytes.Equal(secondContents, secondImage) {
		t.Fatal("replacement did not become the current logo")
	}
	if _, err := application.logoStorage.Open(context.Background(), establishment.ObjectKey(firstKey)); !errors.Is(err, establishment.ErrObjectNotFound) {
		t.Fatalf("replaced object open error = %v, want object removed", err)
	}
}

func TestPutEstablishmentLogoRejectsInvalidAndOversizedMultipartContent(t *testing.T) {
	application := newTestApplication(t)
	registerAndConfirmOwner(t, application, defaultOwnerRegistration())
	for _, test := range []struct {
		name       string
		field      string
		filename   string
		content    []byte
		statusCode int
	}{
		{name: "invalid image", field: "file", filename: "logo.svg", content: []byte("<svg/>"), statusCode: http.StatusBadRequest},
		{name: "wrong form field", field: "image", filename: "logo.png", content: testLogoPNG(t, color.NRGBA{A: 255}), statusCode: http.StatusBadRequest},
		{name: "file limit", field: "file", filename: "logo.png", content: bytes.Repeat([]byte{0x41}, establishment.MaxLogoBytes+1), statusCode: http.StatusRequestEntityTooLarge},
		{name: "multipart envelope limit", field: "file", filename: "logo.png", content: bytes.Repeat([]byte{0x41}, establishment.MaxLogoBytes+90*1024), statusCode: http.StatusRequestEntityTooLarge},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := uploadLogoField(t, application, test.field, test.filename, "image/png", test.content, "")
			assertStatus(t, response, test.statusCode)
		})
	}
	if settings := getEstablishmentSettings(t, application); settings.Logo != nil || settings.IsComplete {
		t.Fatalf("rejected uploads changed establishment profile: %+v", settings)
	}
}

func TestPutEstablishmentLogoRequiresCSRFOwnerAndWritableTrial(t *testing.T) {
	application := newTestApplication(t)
	registerAndConfirmOwner(t, application, defaultOwnerRegistration())
	imageBytes := testLogoPNG(t, color.NRGBA{R: 10, A: 255})

	response := uploadLogo(t, application, "logo.png", "image/png", imageBytes, "invalid-csrf")
	assertStatus(t, response, http.StatusForbidden)
	if settings := getEstablishmentSettings(t, application); settings.Logo != nil {
		t.Fatal("CSRF rejected upload changed profile")
	}

	createProfessionalForList(t, application, "Ana", "ana@example.test")
	if _, err := integrationPool.Exec(context.Background(), `UPDATE users SET password_change_required = false WHERE normalized_email = 'ana@example.test'`); err != nil {
		t.Fatalf("complete professional credentials fixture: %v", err)
	}
	response = application.request(t, http.MethodPost, "/api/v1/auth/login", map[string]string{"email": "ana@example.test", "password": "senha-temporaria"})
	assertStatus(t, response, http.StatusOK)
	response = uploadLogo(t, application, "logo.png", "image/png", imageBytes, "")
	assertStatus(t, response, http.StatusForbidden)
	var forbidden struct {
		Code string `json:"code"`
	}
	decodeJSON(t, response, &forbidden)
	if forbidden.Code != "forbidden" {
		t.Fatalf("professional upload error = %+v", forbidden)
	}

	response = application.request(t, http.MethodPost, "/api/v1/auth/login", map[string]string{"email": defaultOwnerRegistration().Email, "password": defaultOwnerRegistration().Password})
	assertStatus(t, response, http.StatusOK)
	establishmentID := authenticatedEstablishmentID(t, application)
	if _, err := integrationPool.Exec(context.Background(), `UPDATE establishment_trials SET starts_at = $1, ends_at = $2 WHERE establishment_id = $3`, time.Now().Add(-15*24*time.Hour), time.Now().Add(-24*time.Hour), establishmentID); err != nil {
		t.Fatalf("expire trial fixture: %v", err)
	}
	response = uploadLogo(t, application, "logo.png", "image/png", imageBytes, "")
	assertStatus(t, response, http.StatusForbidden)
	var readOnly struct {
		Code string `json:"code"`
	}
	decodeJSON(t, response, &readOnly)
	if readOnly.Code != "read_only" {
		t.Fatalf("expired trial upload error = %+v", readOnly)
	}
	if settings := getEstablishmentSettings(t, application); settings.Logo != nil {
		t.Fatal("expired trial upload changed profile")
	}
}

func TestPutEstablishmentLogoIsTenantScopedAndPreservesCurrentFileWhenAssociationFails(t *testing.T) {
	application := newTestApplication(t)
	firstOwner := defaultOwnerRegistration()
	registerAndConfirmOwner(t, application, firstOwner)
	firstID := authenticatedEstablishmentID(t, application)
	firstImage := testLogoPNG(t, color.NRGBA{R: 20, G: 40, B: 60, A: 255})
	response := uploadLogo(t, application, "logo.png", "image/png", firstImage, "")
	assertStatus(t, response, http.StatusNoContent)
	firstKey, _ := currentLogoContents(t, application, firstID)

	secondOwner := ownerRegistration{EstablishmentName: "Outro Salão", Slug: "outro-salao", TimeZone: "America/Sao_Paulo", Email: "other@example.test", Password: "senha-segura"}
	registerAndConfirmOwner(t, application, secondOwner)
	secondID := authenticatedEstablishmentID(t, application)
	if firstID == secondID {
		t.Fatal("owners unexpectedly share a tenant")
	}
	secondImage := testLogoPNG(t, color.NRGBA{R: 90, G: 120, B: 150, A: 255})
	response = uploadLogo(t, application, "logo.png", "image/png", secondImage, "")
	assertStatus(t, response, http.StatusNoContent)
	if _, contents := currentLogoContents(t, application, firstID); !bytes.Equal(contents, firstImage) {
		t.Fatal("upload to second tenant changed first tenant logo")
	}
	if _, contents := currentLogoContents(t, application, secondID); !bytes.Equal(contents, secondImage) {
		t.Fatal("second tenant did not receive its own logo")
	}

	if _, err := integrationPool.Exec(context.Background(), `CREATE FUNCTION reject_test_logo_metadata() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'test-only logo write failure'; END $$; CREATE TRIGGER reject_test_logo_metadata BEFORE INSERT ON establishment_logos FOR EACH ROW EXECUTE FUNCTION reject_test_logo_metadata()`); err != nil {
		t.Fatalf("create logo association failure trigger: %v", err)
	}
	t.Cleanup(func() {
		_, _ = integrationPool.Exec(context.Background(), `DROP TRIGGER IF EXISTS reject_test_logo_metadata ON establishment_logos; DROP FUNCTION IF EXISTS reject_test_logo_metadata()`)
	})
	replacement := testLogoPNG(t, color.NRGBA{R: 200, G: 180, B: 160, A: 255})
	response = uploadLogo(t, application, "replacement.png", "image/png", replacement, "")
	assertStatus(t, response, http.StatusInternalServerError)
	if _, contents := currentLogoContents(t, application, secondID); !bytes.Equal(contents, secondImage) {
		t.Fatal("failed logo association removed or changed the current logo")
	}
	if _, err := application.logoStorage.Open(context.Background(), firstKey); err != nil {
		t.Fatalf("first tenant object was affected by second tenant upload: %v", err)
	}
}

func TestPutEstablishmentLogoConcurrentReplacementKeepsReferencedObjectReadable(t *testing.T) {
	application := newTestApplication(t)
	registerAndConfirmOwner(t, application, defaultOwnerRegistration())
	images := [][]byte{
		testLogoPNG(t, color.NRGBA{R: 11, G: 22, B: 33, A: 255}),
		testLogoPNG(t, color.NRGBA{R: 44, G: 55, B: 66, A: 255}),
	}
	var wait sync.WaitGroup
	errorsByUpload := make([]error, len(images))
	for index := range images {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			response, err := sendLogoUpload(application, "logo.png", images[index])
			if err != nil {
				errorsByUpload[index] = err
				return
			}
			defer response.Body.Close()
			if response.StatusCode != http.StatusNoContent {
				body, _ := io.ReadAll(response.Body)
				errorsByUpload[index] = errors.New("unexpected status " + response.Status + ": " + string(body))
			}
		}(index)
	}
	wait.Wait()
	for _, err := range errorsByUpload {
		if err != nil {
			t.Fatal(err)
		}
	}
	settings := getEstablishmentSettings(t, application)
	if settings.Logo == nil {
		t.Fatal("concurrent replacements did not associate a current logo")
	}
	_, current := currentLogoContents(t, application, authenticatedEstablishmentID(t, application))
	if !bytes.Equal(current, images[0]) && !bytes.Equal(current, images[1]) {
		t.Fatal("current logo does not match either successful concurrent upload")
	}
}

func uploadLogo(t *testing.T, application *testApplication, filename, contentType string, content []byte, csrf string) *http.Response {
	t.Helper()
	return uploadLogoField(t, application, "file", filename, contentType, content, csrf)
}

func uploadLogoField(t *testing.T, application *testApplication, field, filename, contentType string, content []byte, csrf string) *http.Response {
	t.Helper()
	requestBody := &bytes.Buffer{}
	writer := multipart.NewWriter(requestBody)
	partHeaders := make(textproto.MIMEHeader)
	partHeaders.Set("Content-Disposition", mime.FormatMediaType("form-data", map[string]string{"name": field, "filename": filename}))
	partHeaders.Set("Content-Type", contentType)
	part, err := writer.CreatePart(partHeaders)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodPut, application.server.URL+"/api/v1/establishment/logo", requestBody)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Origin", trustedOrigin)
	if csrf == "" {
		csrf = application.cookie(t, csrfCookieName).Value
	}
	request.Header.Set("X-CSRF-Token", csrf)
	response, err := application.client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })
	return response
}

func sendLogoUpload(application *testApplication, filename string, content []byte) (*http.Response, error) {
	requestBody := &bytes.Buffer{}
	writer := multipart.NewWriter(requestBody)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(content); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	request, err := http.NewRequest(http.MethodPut, application.server.URL+"/api/v1/establishment/logo", requestBody)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Origin", trustedOrigin)
	request.Header.Set("X-CSRF-Token", application.cookieValue(csrfCookieName))
	return application.client.Do(request)
}

func (application *testApplication) cookieValue(name string) string {
	parsed, err := url.Parse(application.server.URL + "/api/v1/auth/logout")
	if err != nil {
		return ""
	}
	for _, cookie := range application.client.Jar.Cookies(parsed) {
		if cookie.Name == name {
			return cookie.Value
		}
	}
	return ""
}

func testLogoPNG(t *testing.T, fill color.NRGBA) []byte {
	t.Helper()
	imageData := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			imageData.Set(x, y, fill)
		}
	}
	var content bytes.Buffer
	if err := png.Encode(&content, imageData); err != nil {
		t.Fatal(err)
	}
	return content.Bytes()
}

func currentLogoContents(t *testing.T, application *testApplication, establishmentID string) (establishment.ObjectKey, []byte) {
	t.Helper()
	parsed, err := uuid.Parse(establishmentID)
	if err != nil {
		t.Fatal(err)
	}
	logo, err := db.New(integrationPool).GetCurrentEstablishmentLogo(context.Background(), pgtype.UUID{Bytes: parsed, Valid: true})
	if err != nil {
		t.Fatalf("query current logo: %v", err)
	}
	object, err := application.logoStorage.Open(context.Background(), establishment.ObjectKey(logo.ObjectKey))
	if err != nil {
		t.Fatalf("open current logo object: %v", err)
	}
	defer object.Close()
	content, err := io.ReadAll(object)
	if err != nil {
		t.Fatalf("read current logo object: %v", err)
	}
	return establishment.ObjectKey(logo.ObjectKey), content
}
