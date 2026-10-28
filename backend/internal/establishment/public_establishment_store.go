package establishment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	db "barbershop-app/backend/db/generated"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type publicEstablishmentQueries interface {
	GetPublicEstablishmentBySlug(context.Context, string) (db.GetPublicEstablishmentBySlugRow, error)
	GetPublicEstablishmentProfile(context.Context, db.GetPublicEstablishmentProfileParams) (db.GetPublicEstablishmentProfileRow, error)
}

type PostgresPublicEstablishmentStore struct {
	queries publicEstablishmentQueries
}

func NewPostgresPublicEstablishmentStore(pool *pgxpool.Pool) *PostgresPublicEstablishmentStore {
	return &PostgresPublicEstablishmentStore{queries: db.New(pool)}
}

func (store *PostgresPublicEstablishmentStore) FindPublicEstablishmentBySlug(ctx context.Context, slug string) (PublicEstablishment, error) {
	row, err := store.queries.GetPublicEstablishmentBySlug(ctx, slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return PublicEstablishment{}, ErrPublicEstablishmentNotFound
	}
	if err != nil {
		return PublicEstablishment{}, fmt.Errorf("query public establishment by slug: %w", err)
	}
	return PublicEstablishment{ID: uuid.UUID(row.ID.Bytes).String(), Name: row.Name, Slug: row.Slug}, nil
}

func (store *PostgresPublicEstablishmentStore) FindPublicEstablishmentProfile(ctx context.Context, establishmentID, slug string) (PublicEstablishmentProfile, error) {
	parsedID, err := uuid.Parse(establishmentID)
	if err != nil {
		return PublicEstablishmentProfile{}, fmt.Errorf("parse public establishment ID: %w", err)
	}
	row, err := store.queries.GetPublicEstablishmentProfile(ctx, db.GetPublicEstablishmentProfileParams{
		EstablishmentID: pgtype.UUID{Bytes: parsedID, Valid: true}, Slug: slug,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return PublicEstablishmentProfile{}, ErrPublicEstablishmentNotFound
	}
	if err != nil {
		return PublicEstablishmentProfile{}, fmt.Errorf("query public establishment profile: %w", err)
	}
	links := make([]SocialLink, 0)
	if err := json.Unmarshal([]byte(row.SocialLinks), &links); err != nil {
		return PublicEstablishmentProfile{}, fmt.Errorf("decode public establishment links: %w", err)
	}
	profile := PublicEstablishmentProfile{Name: row.Name, Slug: row.Slug, SocialLinks: links}
	if row.Description.Valid {
		profile.Description = &row.Description.String
	}
	if row.Address.Valid {
		profile.Address = &row.Address.String
	}
	if row.ContactPhone.Valid {
		profile.ContactPhone = &row.ContactPhone.String
	}
	return profile, nil
}

type publicLogoQueries interface {
	GetPublicEstablishmentLogo(context.Context, db.GetPublicEstablishmentLogoParams) (db.GetPublicEstablishmentLogoRow, error)
}

type PostgresPublicLogoStore struct {
	queries publicLogoQueries
}

func NewPostgresPublicLogoStore(pool *pgxpool.Pool) *PostgresPublicLogoStore {
	return &PostgresPublicLogoStore{queries: db.New(pool)}
}

func (store *PostgresPublicLogoStore) FindPublicLogo(ctx context.Context, establishmentID, slug string) (publicLogoRecord, error) {
	parsedID, err := uuid.Parse(establishmentID)
	if err != nil {
		return publicLogoRecord{}, fmt.Errorf("parse public establishment ID: %w", err)
	}
	logo, err := store.queries.GetPublicEstablishmentLogo(ctx, db.GetPublicEstablishmentLogoParams{
		EstablishmentID: pgtype.UUID{Bytes: parsedID, Valid: true}, Slug: slug,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return publicLogoRecord{}, ErrPublicLogoNotFound
	}
	if err != nil {
		return publicLogoRecord{}, fmt.Errorf("query public establishment logo: %w", err)
	}
	return publicLogoRecord{
		ObjectKey: ObjectKey(logo.ObjectKey), ContentType: logo.ContentType,
		ByteSize: logo.ByteSize, SHA256: logo.ContentSha256,
	}, nil
}
