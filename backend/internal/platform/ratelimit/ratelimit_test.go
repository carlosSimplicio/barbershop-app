package ratelimit

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

var rateLimitTestNow = time.Date(2026, time.October, 18, 15, 0, 0, 0, time.UTC)

func TestLimiterUsesIndependentDimensionsAndLongestProgressiveWindow(t *testing.T) {
	store := &rateStoreStub{events: map[string][]time.Time{
		"email": repeatedEvents(20, time.Hour),
	}}
	limiter := New(store, keyGeneratorStub{})

	_, err := limiter.Reserve(context.Background(), "email-confirmation", map[string]string{
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

func TestLimiterReservesDomainSeparatedKeysForEveryIdentifier(t *testing.T) {
	store := &rateStoreStub{events: map[string][]time.Time{}}
	limiter := New(store, keyGeneratorStub{})

	reservation, err := limiter.Reserve(context.Background(), "register", map[string]string{
		"ip": "192.0.2.1", "email": "owner@example.com",
	}, rateLimitTestNow)
	if err != nil {
		t.Fatalf("Reserve: %v", err)
	}
	if store.recordedScope != "register" || string(store.recorded["ip"]) != "register:ip" || string(store.recorded["email"]) != "register:email" {
		t.Fatalf("recorded keys are not scope separated: scope=%q keys=%q", store.recordedScope, store.recorded)
	}
	if err := limiter.Release(context.Background(), reservation); err != nil || len(store.released) != 2 {
		t.Fatalf("Release: error=%v ids=%v", err, store.released)
	}
}

func TestLimiterPreservesLegacyLoginKeysAcrossMigration(t *testing.T) {
	store := &rateStoreStub{events: map[string][]time.Time{}}
	limiter := New(store, keyGeneratorStub{})

	_, err := limiter.Reserve(context.Background(), "login", map[string]string{
		"ip": "192.0.2.1", "email": "owner@example.com",
	}, rateLimitTestNow)
	if err != nil {
		t.Fatalf("Reserve: %v", err)
	}
	if string(store.recorded["ip"]) != "ip" || string(store.recorded["email"]) != "email" {
		t.Fatalf("login keys do not preserve the pre-migration domain: %q", store.recorded)
	}
}

func TestLimiterAtomicallyRejectsConcurrentAttemptsBeyondThreshold(t *testing.T) {
	store := &concurrentRateStoreStub{events: make(map[string][]time.Time)}
	limiter := New(store, keyGeneratorStub{})
	const attempts = 12
	start := make(chan struct{})
	results := make(chan error, attempts)
	var wait sync.WaitGroup
	for range attempts {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			_, err := limiter.Reserve(context.Background(), "register", map[string]string{"ip": "192.0.2.1"}, rateLimitTestNow)
			results <- err
		}()
	}
	close(start)
	wait.Wait()
	close(results)

	accepted := 0
	for err := range results {
		if err == nil {
			accepted++
			continue
		}
		var rateError Error
		if !errors.As(err, &rateError) {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if accepted != 5 {
		t.Fatalf("accepted attempts = %d, want 5", accepted)
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
	released      []int64
}

func (stub *rateStoreStub) ReserveRateLimitEvents(_ context.Context, scope string, keys map[string][]byte, at time.Time) ([]int64, time.Duration, error) {
	stub.scopes = append(stub.scopes, scope)
	stub.recordedScope = scope
	stub.recorded = keys
	var longest time.Duration
	for dimension := range keys {
		if duration := retryAfter(stub.events[dimension], at); duration > longest {
			longest = duration
		}
	}
	if longest > 0 {
		return nil, longest, nil
	}
	return []int64{1, 2}, 0, nil
}

func (stub *rateStoreStub) ReleaseRateLimitEvents(_ context.Context, ids []int64) error {
	stub.released = append([]int64(nil), ids...)
	return nil
}

type concurrentRateStoreStub struct {
	mu     sync.Mutex
	events map[string][]time.Time
	nextID int64
}

func (stub *concurrentRateStoreStub) ReserveRateLimitEvents(_ context.Context, _ string, keys map[string][]byte, at time.Time) ([]int64, time.Duration, error) {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	for dimension := range keys {
		if duration := retryAfter(stub.events[dimension], at); duration > 0 {
			return nil, duration, nil
		}
	}
	stub.nextID++
	for dimension := range keys {
		stub.events[dimension] = append(stub.events[dimension], at)
	}
	return []int64{stub.nextID}, 0, nil
}

func (*concurrentRateStoreStub) ReleaseRateLimitEvents(context.Context, []int64) error { return nil }
