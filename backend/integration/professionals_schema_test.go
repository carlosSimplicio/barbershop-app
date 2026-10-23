//go:build integration

package integration_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestProfessionalProfileBelongsToUserEstablishment(t *testing.T) {
	resetDatabase(t)
	ctx := context.Background()
	firstEstablishmentID := uuid.New()
	secondEstablishmentID := uuid.New()
	professionalID := uuid.New()

	for _, establishment := range []struct {
		id   uuid.UUID
		slug string
	}{
		{id: firstEstablishmentID, slug: "primeiro-estabelecimento"},
		{id: secondEstablishmentID, slug: "segundo-estabelecimento"},
	} {
		if _, err := integrationPool.Exec(ctx, `
			INSERT INTO establishments (id, name, slug, time_zone)
			VALUES ($1, 'Estabelecimento Fictício', $2, 'America/Sao_Paulo')
		`, establishment.id, establishment.slug); err != nil {
			t.Fatalf("create establishment: %v", err)
		}
	}

	if _, err := integrationPool.Exec(ctx, `
		INSERT INTO users (id, establishment_id, email, normalized_email, role, password_change_required)
		VALUES ($1, $2, 'professional@example.test', 'professional@example.test', 'professional', true)
	`, professionalID, firstEstablishmentID); err != nil {
		t.Fatalf("create professional user: %v", err)
	}

	_, err := integrationPool.Exec(ctx, `
		INSERT INTO professionals (user_id, establishment_id, name)
		VALUES ($1, $2, 'Profissional Fictício')
	`, professionalID, secondEstablishmentID)
	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) || postgresError.ConstraintName != "professionals_user_establishment_fk" {
		t.Fatalf("cross-establishment profile error = %v, want professionals_user_establishment_fk", err)
	}

	if _, err := integrationPool.Exec(ctx, `
		INSERT INTO professionals (user_id, establishment_id, name)
		VALUES ($1, $2, 'Profissional Fictício')
	`, professionalID, firstEstablishmentID); err != nil {
		t.Fatalf("create professional profile in same establishment: %v", err)
	}
}
