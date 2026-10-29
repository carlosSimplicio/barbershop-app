package establishment

import (
	"context"
	"encoding/json"
	"fmt"

	db "barbershop-app/backend/db/generated"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type establishmentSettingsQueries interface {
	GetEstablishmentSettingsByID(context.Context, pgtype.UUID) (db.GetEstablishmentSettingsByIDRow, error)
}

type PostgresSettingsStore struct {
	queries establishmentSettingsQueries
	pool    *pgxpool.Pool
}

func NewPostgresSettingsStore(pool *pgxpool.Pool) *PostgresSettingsStore {
	return &PostgresSettingsStore{queries: db.New(pool), pool: pool}
}

func (store *PostgresSettingsStore) UpdateSettings(ctx context.Context, establishmentID string, patch SettingsPatch) error {
	parsedID, err := uuid.Parse(establishmentID)
	if err != nil {
		return fmt.Errorf("parse establishment ID: %w", err)
	}
	establishmentUUID := pgtype.UUID{Bytes: parsedID, Valid: true}
	transaction, err := store.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin establishment settings update: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	transactionQueries := db.New(transaction)
	lockedEstablishment, err := transactionQueries.LockEstablishmentOperationalWrite(ctx, establishmentUUID)
	if err != nil {
		return fmt.Errorf("lock establishment for settings update: %w", err)
	}
	if patch.TimeZone.Set && patch.TimeZone.Value != lockedEstablishment.TimeZone && lockedEstablishment.OperationalDataStartedAt.Valid {
		return ErrTimeZoneLocked
	}
	rowsAffected, err := transactionQueries.PatchEstablishmentSettings(ctx, db.PatchEstablishmentSettingsParams{
		NameSet: patch.Name.Set, Name: patch.Name.Value,
		TimeZoneSet: patch.TimeZone.Set, TimeZone: patch.TimeZone.Value,
		DescriptionSet: patch.Description.Set, Description: pgtype.Text{String: patch.Description.Value, Valid: patch.Description.Set && !patch.Description.Null},
		AddressSet: patch.Address.Set, Address: pgtype.Text{String: patch.Address.Value, Valid: patch.Address.Set && !patch.Address.Null},
		ContactPhoneSet: patch.ContactPhone.Set, ContactPhone: pgtype.Text{String: patch.ContactPhone.Value, Valid: patch.ContactPhone.Set && !patch.ContactPhone.Null},
		MinimumBookingNoticeMinutesSet:   patch.MinimumBookingNoticeMinutes.Set,
		MinimumBookingNoticeMinutes:      pgtype.Int4{Int32: patch.MinimumBookingNoticeMinutes.Value, Valid: patch.MinimumBookingNoticeMinutes.Set && !patch.MinimumBookingNoticeMinutes.Null},
		BookingHorizonDaysSet:            patch.BookingHorizonDays.Set,
		BookingHorizonDays:               pgtype.Int4{Int32: patch.BookingHorizonDays.Value, Valid: patch.BookingHorizonDays.Set && !patch.BookingHorizonDays.Null},
		CustomerChangeDeadlineMinutesSet: patch.CustomerChangeDeadlineMinutes.Set,
		CustomerChangeDeadlineMinutes:    pgtype.Int4{Int32: patch.CustomerChangeDeadlineMinutes.Value, Valid: patch.CustomerChangeDeadlineMinutes.Set && !patch.CustomerChangeDeadlineMinutes.Null},
		EstablishmentID:                  establishmentUUID,
	})
	if err != nil {
		return fmt.Errorf("update establishment settings: %w", err)
	}
	if rowsAffected != 1 {
		return fmt.Errorf("update establishment settings: expected one row, got %d", rowsAffected)
	}
	if patch.SocialLinks.Set {
		if err := transactionQueries.DeleteEstablishmentSocialLinks(ctx, establishmentUUID); err != nil {
			return fmt.Errorf("replace establishment social links: %w", err)
		}
		for position, link := range patch.SocialLinks.Value {
			if err := transactionQueries.CreateEstablishmentSocialLink(ctx, db.CreateEstablishmentSocialLinkParams{
				EstablishmentID: establishmentUUID, Position: int32(position), Label: link.Label, Url: link.URL,
			}); err != nil {
				return fmt.Errorf("create establishment social link: %w", err)
			}
		}
	}
	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("commit establishment settings update: %w", err)
	}
	return nil
}

func (store *PostgresSettingsStore) GetSettingsByID(ctx context.Context, establishmentID string) (EstablishmentSettings, error) {
	parsedID, err := uuid.Parse(establishmentID)
	if err != nil {
		return EstablishmentSettings{}, fmt.Errorf("parse establishment ID: %w", err)
	}
	establishmentUUID := pgtype.UUID{Bytes: parsedID, Valid: true}
	row, err := store.queries.GetEstablishmentSettingsByID(ctx, establishmentUUID)
	if err != nil {
		return EstablishmentSettings{}, fmt.Errorf("query establishment settings: %w", err)
	}
	links := make([]SocialLink, 0)
	if err := json.Unmarshal([]byte(row.SocialLinks), &links); err != nil {
		return EstablishmentSettings{}, fmt.Errorf("decode establishment social links: %w", err)
	}
	result := EstablishmentSettings{
		Establishment: Establishment{ID: uuid.UUID(row.ID.Bytes).String(), Name: row.Name, Slug: row.Slug, TimeZone: row.TimeZone},
		Description:   nullableText(row.Description), Address: nullableText(row.Address), ContactPhone: nullableText(row.ContactPhone),
		MinimumBookingNoticeMinutes:   nullableInt32(row.MinimumBookingNoticeMinutes),
		BookingHorizonDays:            nullableInt32(row.BookingHorizonDays),
		CustomerChangeDeadlineMinutes: nullableInt32(row.CustomerChangeDeadlineMinutes),
		OperationalDataStarted:        row.OperationalDataStartedAt.Valid,
		SocialLinks:                   links,
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
