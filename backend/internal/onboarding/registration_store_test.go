package onboarding

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestCreateRegistrationRollsBackWhenAnyWriteFails(t *testing.T) {
	writes := []string{"establishment", "owner", "credential", "confirmation"}
	for _, failedWrite := range writes {
		t.Run(failedWrite, func(t *testing.T) {
			transaction := &registrationTransactionStub{failAt: failedWrite}
			store := &Store{beginRegistration: func(context.Context) (registrationTransaction, error) {
				return transaction, nil
			}}

			if err := store.CreateRegistration(context.Background(), Registration{}); err == nil {
				t.Fatal("CreateRegistration error = nil")
			}
			if !transaction.rolledBack || transaction.committed {
				t.Fatalf("rolledBack = %t, committed = %t", transaction.rolledBack, transaction.committed)
			}
			failedIndex := 0
			for index, write := range writes {
				if write == failedWrite {
					failedIndex = index
				}
			}
			if want := writes[:failedIndex+1]; !reflect.DeepEqual(transaction.writes, want) {
				t.Fatalf("writes = %#v, want %#v", transaction.writes, want)
			}
		})
	}
}

func TestCreateRegistrationCommitsOnlyAfterAllWrites(t *testing.T) {
	transaction := &registrationTransactionStub{}
	store := &Store{beginRegistration: func(context.Context) (registrationTransaction, error) { return transaction, nil }}

	if err := store.CreateRegistration(context.Background(), Registration{}); err != nil {
		t.Fatalf("CreateRegistration: %v", err)
	}
	if !transaction.committed || !reflect.DeepEqual(transaction.writes, []string{"establishment", "owner", "credential", "confirmation"}) {
		t.Fatalf("committed = %t, writes = %#v", transaction.committed, transaction.writes)
	}
}

func TestRegistrationWriteErrorMapsOnlyExpectedFieldConflicts(t *testing.T) {
	tests := []struct {
		constraint string
		want       error
	}{
		{constraint: "users_normalized_email_unique", want: ErrEmailUnavailable},
		{constraint: "establishments_slug_unique", want: ErrSlugUnavailable},
	}
	for _, test := range tests {
		err := registrationWriteError("write", &pgconn.PgError{ConstraintName: test.constraint})
		if !errors.Is(err, test.want) {
			t.Fatalf("constraint %q: error = %v, want %v", test.constraint, err, test.want)
		}
	}
}

type registrationTransactionStub struct {
	failAt     string
	writes     []string
	committed  bool
	rolledBack bool
}

func (stub *registrationTransactionStub) write(name string) error {
	stub.writes = append(stub.writes, name)
	if stub.failAt == name {
		return errors.New("write failed")
	}
	return nil
}

func (stub *registrationTransactionStub) CreateEstablishment(context.Context, Registration) (pgtype.UUID, error) {
	return pgtype.UUID{Valid: true}, stub.write("establishment")
}

func (stub *registrationTransactionStub) CreateOwner(context.Context, pgtype.UUID, Registration) (pgtype.UUID, error) {
	return pgtype.UUID{Valid: true}, stub.write("owner")
}

func (stub *registrationTransactionStub) CreateCredential(context.Context, pgtype.UUID, Registration) error {
	return stub.write("credential")
}

func (stub *registrationTransactionStub) CreateConfirmation(context.Context, pgtype.UUID, Registration) error {
	return stub.write("confirmation")
}

func (stub *registrationTransactionStub) Commit(context.Context) error {
	stub.committed = true
	return nil
}

func (stub *registrationTransactionStub) Rollback(context.Context) error {
	stub.rolledBack = true
	return nil
}
