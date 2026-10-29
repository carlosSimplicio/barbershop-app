package ratelimit

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"
	"time"

	db "barbershop-app/backend/db/generated"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool, queries: db.New(pool)}
}

func (store *PostgresStore) ReserveRateLimitEvents(ctx context.Context, scope string, keys map[string][]byte, attemptedAt time.Time) ([]int64, time.Duration, error) {
	transaction, err := store.pool.Begin(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("begin rate limit event transaction: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	queries := store.queries.WithTx(transaction)
	dimensions := make([]string, 0, len(keys))
	for dimension := range keys {
		dimensions = append(dimensions, dimension)
	}
	sort.Strings(dimensions)

	for _, dimension := range dimensions {
		if _, err := transaction.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", advisoryLockID(scope, dimension, keys[dimension])); err != nil {
			return nil, 0, fmt.Errorf("lock rate limit key: %w", err)
		}
	}

	var longest time.Duration
	for _, dimension := range dimensions {
		rows, err := queries.ListRateLimitEventsSince(ctx, db.ListRateLimitEventsSinceParams{
			Scope: scope, Dimension: dimension, KeyHash: keys[dimension], AttemptedAt: rateLimitTimestamp(attemptedAt.Add(-24 * time.Hour)),
		})
		if err != nil {
			return nil, 0, fmt.Errorf("list rate limit events: %w", err)
		}
		events := make([]time.Time, 0, len(rows))
		for _, row := range rows {
			events = append(events, row.Time)
		}
		if duration := retryAfter(events, attemptedAt); duration > longest {
			longest = duration
		}
	}
	if longest > 0 {
		return nil, longest, nil
	}

	eventIDs := make([]int64, 0, len(dimensions))
	for _, dimension := range dimensions {
		eventID, err := queries.RecordRateLimitEvent(ctx, db.RecordRateLimitEventParams{
			Scope: scope, Dimension: dimension, KeyHash: keys[dimension], AttemptedAt: rateLimitTimestamp(attemptedAt),
		})
		if err != nil {
			return nil, 0, fmt.Errorf("record rate limit event: %w", err)
		}
		eventIDs = append(eventIDs, eventID)
	}
	if err := transaction.Commit(ctx); err != nil {
		return nil, 0, fmt.Errorf("commit rate limit event transaction: %w", err)
	}
	return eventIDs, 0, nil
}

func (store *PostgresStore) ReleaseRateLimitEvents(ctx context.Context, eventIDs []int64) error {
	if len(eventIDs) == 0 {
		return nil
	}
	if err := store.queries.DeleteRateLimitEventsByIDs(ctx, eventIDs); err != nil {
		return fmt.Errorf("release rate limit events: %w", err)
	}
	return nil
}

func advisoryLockID(scope, dimension string, keyHash []byte) int64 {
	digest := sha256.New()
	_, _ = digest.Write([]byte(scope))
	_, _ = digest.Write([]byte{0})
	_, _ = digest.Write([]byte(dimension))
	_, _ = digest.Write([]byte{0})
	_, _ = digest.Write(keyHash)
	return int64(binary.BigEndian.Uint64(digest.Sum(nil)))
}

func rateLimitTimestamp(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value.UTC(), Valid: true}
}

var _ Store = (*PostgresStore)(nil)
