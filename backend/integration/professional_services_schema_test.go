//go:build integration

package integration_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestProfessionalServicesSchemaProtectsPairAndTenant(t *testing.T) {
	resetDatabase(t)
	ctx := context.Background()
	tenant, other := uuid.New(), uuid.New()
	professionalID, otherProfessionalID, offeringID, otherOfferingID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	for index, establishmentID := range []uuid.UUID{tenant, other} {
		if _, err := integrationPool.Exec(ctx, `INSERT INTO establishments (id,name,slug,time_zone) VALUES ($1,'Estabelecimento fictício',$2,'America/Sao_Paulo')`, establishmentID, establishmentID.String()); err != nil {
			t.Fatal(err)
		}
		userID := []uuid.UUID{professionalID, otherProfessionalID}[index]
		serviceID := []uuid.UUID{offeringID, otherOfferingID}[index]
		if _, err := integrationPool.Exec(ctx, `INSERT INTO users (id,establishment_id,email,normalized_email,role) VALUES ($1,$2,$3,$3,'professional')`, userID, establishmentID, userID.String()+"@example.test"); err != nil {
			t.Fatal(err)
		}
		if _, err := integrationPool.Exec(ctx, `INSERT INTO professionals (user_id,establishment_id,name) VALUES ($1,$2,'Profissional fictício')`, userID, establishmentID); err != nil {
			t.Fatal(err)
		}
		if _, err := integrationPool.Exec(ctx, `INSERT INTO services (id,establishment_id,name,price_cents,duration_minutes) VALUES ($1,$2,'Serviço fictício',1250,30)`, serviceID, establishmentID); err != nil {
			t.Fatal(err)
		}
	}
	var count int
	if err := integrationPool.QueryRow(ctx, `SELECT count(*) FROM professional_services`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("automatic associations=%d err=%v", count, err)
	}
	var pairID uuid.UUID
	if err := integrationPool.QueryRow(ctx, `INSERT INTO professional_services (establishment_id,professional_id,service_id) VALUES ($1,$2,$3) RETURNING id`, tenant, professionalID, offeringID).Scan(&pairID); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name                           string
		tenant, professional, offering uuid.UUID
		code                           string
	}{
		{"duplicate", tenant, professionalID, offeringID, "23505"},
		{"foreign professional", tenant, otherProfessionalID, offeringID, "23503"},
		{"foreign service", tenant, professionalID, otherOfferingID, "23503"},
		{"missing professional", tenant, uuid.New(), offeringID, "23503"},
		{"missing service", tenant, professionalID, uuid.New(), "23503"},
		{"missing tenant", uuid.New(), professionalID, offeringID, "23503"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := integrationPool.Exec(ctx, `INSERT INTO professional_services (establishment_id,professional_id,service_id) VALUES ($1,$2,$3)`, test.tenant, test.professional, test.offering)
			var postgresError *pgconn.PgError
			if !errors.As(err, &postgresError) || postgresError.Code != test.code {
				t.Fatalf("error=%v want SQLSTATE %s", err, test.code)
			}
		})
	}
	if _, err := integrationPool.Exec(ctx, `UPDATE professional_services SET active=false WHERE id=$1`, pairID); err != nil {
		t.Fatal(err)
	}
	if _, err := integrationPool.Exec(ctx, `UPDATE professional_services SET active=true WHERE id=$1`, pairID); err != nil {
		t.Fatal(err)
	}
	var retainedID uuid.UUID
	var active, started bool
	if err := integrationPool.QueryRow(ctx, `SELECT id,active FROM professional_services WHERE establishment_id=$1 AND professional_id=$2 AND service_id=$3`, tenant, professionalID, offeringID).Scan(&retainedID, &active); err != nil || retainedID != pairID || !active {
		t.Fatalf("pair not preserved: %v %v %v", retainedID, active, err)
	}
	for _, statement := range []string{`DELETE FROM professionals WHERE user_id=$1`, `DELETE FROM services WHERE id=$1`} {
		targetID := professionalID
		if statement == `DELETE FROM services WHERE id=$1` {
			targetID = offeringID
		}
		_, err := integrationPool.Exec(ctx, statement, targetID)
		var postgresError *pgconn.PgError
		if !errors.As(err, &postgresError) || postgresError.Code != "23001" {
			t.Fatalf("destructive delete error=%v", err)
		}
	}
	if err := integrationPool.QueryRow(ctx, `SELECT operational_data_started_at IS NOT NULL FROM establishments WHERE id=$1`, tenant).Scan(&started); err != nil || started {
		t.Fatalf("operational marker changed: %v %v", started, err)
	}
}
