//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptrace"
	"sync"
	"testing"
	"time"
)

type inflightPasswordResponse struct {
	response *http.Response
	err      error
}

func TestProfessionalReactivationRejectsPasswordChangeAuthenticatedBeforeDeactivation(t *testing.T) {
	app := newTestApplication(t)
	id := createManagementProfessional(t, app)
	peer := independentClient(t, app)
	loginProfessional(t, peer)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	body := []byte(`{"new_password":"senha-antiga-em-voo"}`)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, app.server.URL+"/api/v1/auth/password/change", reader)
	if err != nil {
		t.Fatal(err)
	}
	request.ContentLength = int64(len(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", trustedOrigin)
	request.Header.Set("Expect", "100-continue")
	csrf := peer.cookie(t, csrfCookieName)
	request.AddCookie(csrf)
	request.Header.Set("X-CSRF-Token", csrf.Value)
	request.AddCookie(peer.cookie(t, accessCookieName))
	bodyRequested := make(chan struct{})
	var once sync.Once
	request = request.WithContext(httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{Got100Continue: func() { once.Do(func() { close(bodyRequested) }) }}))
	responses := make(chan inflightPasswordResponse, 1)
	go func() {
		response, err := (&http.Client{Transport: peer.client.Transport}).Do(request)
		responses <- inflightPasswordResponse{response, err}
	}()
	// net/http sends 100 Continue only when the handler reads the body. Since
	// authentication runs first, this gates the request after its old principal.
	select {
	case <-bodyRequested:
	case result := <-responses:
		if result.response != nil {
			_ = result.response.Body.Close()
		}
		t.Fatalf("password handler did not reach body read: %v", result.err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	deactivateManagementProfessional(t, app, id)
	assertStatus(t, app.request(t, http.MethodPost, "/api/v1/professionals/"+id+"/reactivation", map[string]string{"temporary_password": "nova-temporaria"}), http.StatusOK)
	before := readProfessionalSecurityState(t, id)
	if _, err = io.Copy(writer, bytes.NewReader(body)); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case result := <-responses:
		if result.err != nil {
			t.Fatal(result.err)
		}
		defer result.response.Body.Close()
		assertStatus(t, result.response, http.StatusForbidden)
		for _, cookie := range result.response.Cookies() {
			if cookie.MaxAge > 0 {
				t.Fatal("old authenticated request issued a valid session cookie")
			}
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	if readProfessionalSecurityState(t, id) != before {
		t.Fatal("old authenticated request replaced the reactivated credential")
	}
	assertStatus(t, peer.request(t, http.MethodGet, "/api/v1/auth/me", nil), http.StatusUnauthorized)
	fresh := independentClient(t, app)
	assertStatus(t, fresh.request(t, http.MethodPost, "/api/v1/auth/login", map[string]string{"email": "ana@example.test", "password": "nova-temporaria"}), http.StatusOK)
}
