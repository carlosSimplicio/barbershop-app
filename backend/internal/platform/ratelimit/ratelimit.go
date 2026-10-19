package ratelimit

import (
	"context"
	"sort"
	"time"
)

type Store interface {
	ReserveRateLimitEvents(context.Context, string, map[string][]byte, time.Time) ([]int64, time.Duration, error)
	ReleaseRateLimitEvents(context.Context, []int64) error
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

type Reservation struct {
	eventIDs []int64
}

func (limiter *Limiter) Reserve(ctx context.Context, scope string, identifiers map[string]string, at time.Time) (*Reservation, error) {
	keys := make(map[string][]byte, len(identifiers))
	for dimension, value := range identifiers {
		keys[dimension] = limiter.keys.Key(keyDomain(scope, dimension), value)
	}
	eventIDs, retryAfter, err := limiter.store.ReserveRateLimitEvents(ctx, scope, keys, at)
	if err != nil {
		return nil, err
	}
	if retryAfter > 0 {
		return nil, Error{RetryAfter: retryAfter}
	}
	return &Reservation{eventIDs: eventIDs}, nil
}

func (limiter *Limiter) Release(ctx context.Context, reservation *Reservation) error {
	if reservation == nil || len(reservation.eventIDs) == 0 {
		return nil
	}
	return limiter.store.ReleaseRateLimitEvents(ctx, reservation.eventIDs)
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
