package ratelimit

import (
	"context"
	"fmt"
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

func (store *PostgresStore) ListRateLimitEvents(ctx context.Context, scope, dimension string, keyHash []byte, since time.Time) ([]time.Time, error) {
	rows, err := store.queries.ListRateLimitEventsSince(ctx, db.ListRateLimitEventsSinceParams{
		Scope: scope, Dimension: dimension, KeyHash: keyHash, AttemptedAt: rateLimitTimestamp(since),
	})
	if err != nil {
		return nil, fmt.Errorf("list rate limit events: %w", err)
	}
	events := make([]time.Time, 0, len(rows))
	for _, row := range rows {
		events = append(events, row.Time)
	}
	return events, nil
}

func (store *PostgresStore) RecordRateLimitEvents(ctx context.Context, scope string, keys map[string][]byte, attemptedAt time.Time) error {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin rate limit event transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := store.queries.WithTx(tx)
	for dimension, key := range keys {
		if err := queries.RecordRateLimitEvent(ctx, db.RecordRateLimitEventParams{
			Scope: scope, Dimension: dimension, KeyHash: key, AttemptedAt: rateLimitTimestamp(attemptedAt),
		}); err != nil {
			return fmt.Errorf("record rate limit event: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit rate limit event transaction: %w", err)
	}
	return nil
}

func rateLimitTimestamp(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value.UTC(), Valid: true}
}

var _ Store = (*PostgresStore)(nil)
