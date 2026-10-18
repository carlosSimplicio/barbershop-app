package ratelimit

import (
	"context"
	"errors"
	"testing"
	"time"
)

var rateLimitTestNow = time.Date(2026, time.October, 18, 15, 0, 0, 0, time.UTC)

func TestLimiterUsesIndependentDimensionsAndLongestProgressiveWindow(t *testing.T) {
	store := &rateStoreStub{events: map[string][]time.Time{
		"email": repeatedEvents(20, time.Hour),
	}}
	limiter := New(store, keyGeneratorStub{})

	err := limiter.Check(context.Background(), "email-confirmation", map[string]string{
		"ip": "192.0.2.1", "email": "owner@example.com",
	}, rateLimitTestNow)
	var rateError Error
	if !errors.As(err, &rateError) {
		t.Fatalf("error = %v, want rate limit Error", err)
	}
	if rateError.RetryAfter != 4*time.Hour {
		t.Fatalf("RetryAfter = %s, want 4h", rateError.RetryAfter)
	}
	if store.scopes[0] != "email-confirmation" {
		t.Fatalf("scope = %q", store.scopes[0])
	}
}

func TestLimiterRecordsDomainSeparatedKeysForEveryIdentifier(t *testing.T) {
	store := &rateStoreStub{events: map[string][]time.Time{}}
	limiter := New(store, keyGeneratorStub{})

	if err := limiter.Record(context.Background(), "register", map[string]string{
		"ip": "192.0.2.1", "email": "owner@example.com",
	}, rateLimitTestNow); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if store.recordedScope != "register" || string(store.recorded["ip"]) != "register:ip" || string(store.recorded["email"]) != "register:email" {
		t.Fatalf("recorded keys are not scope separated: scope=%q keys=%q", store.recordedScope, store.recorded)
	}
}

func TestLimiterPreservesLegacyLoginKeysAcrossMigration(t *testing.T) {
	store := &rateStoreStub{events: map[string][]time.Time{}}
	limiter := New(store, keyGeneratorStub{})

	if err := limiter.Record(context.Background(), "login", map[string]string{
		"ip": "192.0.2.1", "email": "owner@example.com",
	}, rateLimitTestNow); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if string(store.recorded["ip"]) != "ip" || string(store.recorded["email"]) != "email" {
		t.Fatalf("login keys do not preserve the pre-migration domain: %q", store.recorded)
	}
}

func repeatedEvents(count int, distance time.Duration) []time.Time {
	events := make([]time.Time, count)
	for index := range events {
		events[index] = rateLimitTestNow.Add(-time.Duration(index+1) * distance)
	}
	return events
}

type keyGeneratorStub struct{}

func (keyGeneratorStub) Key(dimension, _ string) []byte { return []byte(dimension) }

type rateStoreStub struct {
	events        map[string][]time.Time
	scopes        []string
	recordedScope string
	recorded      map[string][]byte
}

func (stub *rateStoreStub) ListRateLimitEvents(_ context.Context, scope, dimension string, _ []byte, _ time.Time) ([]time.Time, error) {
	stub.scopes = append(stub.scopes, scope)
	return stub.events[dimension], nil
}

func (stub *rateStoreStub) RecordRateLimitEvents(_ context.Context, scope string, keys map[string][]byte, _ time.Time) error {
	stub.recordedScope = scope
	stub.recorded = keys
	return nil
}
