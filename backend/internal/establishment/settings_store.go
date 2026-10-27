package establishment

import (
	"context"
	"fmt"

	db "barbershop-app/backend/db/generated"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type establishmentSettingsQueries interface {
	GetEstablishmentSettingsByID(context.Context, pgtype.UUID) (db.GetEstablishmentSettingsByIDRow, error)
	ListEstablishmentSocialLinks(context.Context, pgtype.UUID) ([]db.EstablishmentSocialLink, error)
}

type PostgresSettingsStore struct {
	queries establishmentSettingsQueries
}

func NewPostgresSettingsStore(pool *pgxpool.Pool) *PostgresSettingsStore {
	return &PostgresSettingsStore{queries: db.New(pool)}
}

func (store *PostgresSettingsStore) GetSettingsByID(ctx context.Context, establishmentID string) (EstablishmentSettings, error) {
	parsedID, err := uuid.Parse(establishmentID)
	if err != nil {
		return EstablishmentSettings{}, fmt.Errorf("parse establishment ID: %w", err)
	}
	queryID := pgtype.UUID{Bytes: parsedID, Valid: true}
	row, err := store.queries.GetEstablishmentSettingsByID(ctx, queryID)
	if err != nil {
		return EstablishmentSettings{}, fmt.Errorf("query establishment settings: %w", err)
	}
	links, err := store.queries.ListEstablishmentSocialLinks(ctx, queryID)
	if err != nil {
		return EstablishmentSettings{}, fmt.Errorf("query establishment social links: %w", err)
	}
	result := EstablishmentSettings{
		Establishment: Establishment{ID: uuid.UUID(row.ID.Bytes).String(), Name: row.Name, Slug: row.Slug, TimeZone: row.TimeZone},
		Description:   nullableText(row.Description), Address: nullableText(row.Address), ContactPhone: nullableText(row.ContactPhone),
		MinimumBookingNoticeMinutes:   nullableInt32(row.MinimumBookingNoticeMinutes),
		BookingHorizonDays:            nullableInt32(row.BookingHorizonDays),
		CustomerChangeDeadlineMinutes: nullableInt32(row.CustomerChangeDeadlineMinutes),
		OperationalDataStarted:        row.OperationalDataStartedAt.Valid,
		SocialLinks:                   make([]SocialLink, 0, len(links)),
	}
	for _, link := range links {
		result.SocialLinks = append(result.SocialLinks, SocialLink{Label: link.Label, URL: link.Url})
	}
	logoFields := []bool{row.LogoContentType.Valid, row.LogoByteSize.Valid, row.LogoWidth.Valid, row.LogoHeight.Valid}
	logoCount := 0
	for _, valid := range logoFields {
		if valid {
			logoCount++
		}
	}
	if logoCount != 0 && logoCount != len(logoFields) {
		return EstablishmentSettings{}, fmt.Errorf("query establishment settings: inconsistent logo metadata")
	}
	if logoCount == len(logoFields) {
		result.Logo = &LogoMetadata{
			ContentType: row.LogoContentType.String, ByteSize: row.LogoByteSize.Int64,
			Width: row.LogoWidth.Int32, Height: row.LogoHeight.Int32,
		}
	}
	return result, nil
}

func nullableText(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	result := value.String
	return &result
}

func nullableInt32(value pgtype.Int4) *int32 {
	if !value.Valid {
		return nil
	}
	result := value.Int32
	return &result
}
