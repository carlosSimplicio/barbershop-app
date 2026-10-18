package ratelimit

import (
	"context"
	"sort"
	"time"
)

type Store interface {
	ListRateLimitEvents(context.Context, string, string, []byte, time.Time) ([]time.Time, error)
	RecordRateLimitEvents(context.Context, string, map[string][]byte, time.Time) error
}

type KeyGenerator interface {
	Key(string, string) []byte
}

type Error struct {
	RetryAfter time.Duration
}

func (err Error) Error() string { return "rate limited" }

type Limiter struct {
	store Store
	keys  KeyGenerator
}

func New(store Store, keys KeyGenerator) *Limiter {
	return &Limiter{store: store, keys: keys}
}

func (limiter *Limiter) Check(ctx context.Context, scope string, identifiers map[string]string, now time.Time) error {
	var longest time.Duration
	for dimension, value := range identifiers {
		events, err := limiter.store.ListRateLimitEvents(
			ctx, scope, dimension, limiter.keys.Key(keyDomain(scope, dimension), value), now.Add(-24*time.Hour),
		)
		if err != nil {
			return err
		}
		if retryAfter := retryAfter(events, now); retryAfter > longest {
			longest = retryAfter
		}
	}
	if longest > 0 {
		return Error{RetryAfter: longest}
	}
	return nil
}

func (limiter *Limiter) Record(ctx context.Context, scope string, identifiers map[string]string, at time.Time) error {
	keys := make(map[string][]byte, len(identifiers))
	for dimension, value := range identifiers {
		keys[dimension] = limiter.keys.Key(keyDomain(scope, dimension), value)
	}
	return limiter.store.RecordRateLimitEvents(ctx, scope, keys, at)
}

func keyDomain(scope, dimension string) string {
	if scope == "login" {
		return dimension
	}
	return scope + ":" + dimension
}

func retryAfter(events []time.Time, now time.Time) time.Duration {
	rules := []struct {
		threshold int
		window    time.Duration
	}{{5, 15 * time.Minute}, {10, time.Hour}, {20, 24 * time.Hour}}
	sorted := append([]time.Time(nil), events...)
	sort.Slice(sorted, func(left, right int) bool { return sorted[left].After(sorted[right]) })
	var longest time.Duration
	for _, rule := range rules {
		cutoff := now.Add(-rule.window)
		recent := make([]time.Time, 0, len(sorted))
		for _, event := range sorted {
			if !event.Before(cutoff) {
				recent = append(recent, event)
			}
		}
		if len(recent) < rule.threshold {
			continue
		}
		if duration := recent[rule.threshold-1].Add(rule.window).Sub(now); duration > longest {
			longest = duration
		}
	}
	return longest
}
