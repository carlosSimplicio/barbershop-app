package establishment

import (
	"context"
	"errors"
	"fmt"

	db "barbershop-app/backend/db/generated"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresLogoUploadStore struct {
	pool *pgxpool.Pool
}

func NewPostgresLogoUploadStore(pool *pgxpool.Pool) *PostgresLogoUploadStore {
	return &PostgresLogoUploadStore{pool: pool}
}

func (store *PostgresLogoUploadStore) ReplaceCurrentLogo(ctx context.Context, establishmentID string, key ObjectKey, logo ValidatedLogo) (ObjectKey, error) {
	parsedID, err := uuid.Parse(establishmentID)
	if err != nil {
		return "", fmt.Errorf("parse establishment ID: %w", err)
	}
	establishmentUUID := pgtype.UUID{Bytes: parsedID, Valid: true}
	transaction, err := store.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin logo association: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	transactionQueries := db.New(transaction)
	if _, err := transactionQueries.LockEstablishmentOperationalWrite(ctx, establishmentUUID); err != nil {
		return "", fmt.Errorf("lock establishment for logo update: %w", err)
	}
	previousKey := ObjectKey("")
	previous, err := transactionQueries.GetCurrentEstablishmentLogo(ctx, establishmentUUID)
	if err == nil {
		previousKey = ObjectKey(previous.ObjectKey)
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("get current establishment logo: %w", err)
	}
	metadata, err := transactionQueries.CreateEstablishmentLogo(ctx, db.CreateEstablishmentLogoParams{
		EstablishmentID: establishmentUUID, ObjectKey: string(key), ContentType: logo.MediaType,
		ByteSize: logo.SizeBytes, Width: int32(logo.Width), Height: int32(logo.Height), ContentSha256: logo.SHA256[:],
	})
	if err != nil {
		return previousKey, fmt.Errorf("create establishment logo metadata: %w", err)
	}
	rows, err := transactionQueries.SetCurrentEstablishmentLogo(ctx, db.SetCurrentEstablishmentLogoParams{
		LogoID: metadata.ID, EstablishmentID: establishmentUUID,
	})
	if err != nil {
		return previousKey, fmt.Errorf("associate establishment logo: %w", err)
	}
	if rows != 1 {
		return previousKey, fmt.Errorf("associate establishment logo: expected one row, got %d", rows)
	}
	if err := transaction.Commit(ctx); err != nil {
		return previousKey, fmt.Errorf("commit establishment logo association: %w", err)
	}
	return previousKey, nil
}

func (store *PostgresLogoUploadStore) CurrentLogoObjectKey(ctx context.Context, establishmentID string) (ObjectKey, error) {
	parsedID, err := uuid.Parse(establishmentID)
	if err != nil {
		return "", fmt.Errorf("parse establishment ID: %w", err)
	}
	logo, err := db.New(store.pool).GetCurrentEstablishmentLogo(ctx, pgtype.UUID{Bytes: parsedID, Valid: true})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get current establishment logo key: %w", err)
	}
	return ObjectKey(logo.ObjectKey), nil
}
