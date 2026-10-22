package onboarding

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"barbershop-app/backend/internal/auth"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

const testSessionID = "018f47d2-e886-7f1c-8e8d-8b38eaf16466"

func TestConfirmEmailAndCreateSessionCommitsExactTrialAndSession(t *testing.T) {
	transaction := validVerificationTransaction(t)
	store := testVerificationStore(transaction)

	var user auth.User
	err := store.ConfirmEmailAndCreateSession(context.Background(), []byte("confirmation"), testSessionID, registrationTestNow, registrationTestNow.Add(15*24*time.Hour), func(confirmed auth.User) error {
		user = confirmed
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(transaction.operations, []string{"lock", "confirm", "consume", "trial", "session", "commit"}) {
		t.Fatalf("operations = %#v", transaction.operations)
	}
	if !transaction.trial.StartsAt.Equal(registrationTestNow) || !transaction.trial.EndsAt.Equal(registrationTestNow.Add(14*24*time.Hour)) {
		t.Fatalf("trial = %#v", transaction.trial)
	}
	if user.Role != "owner" || !user.EmailConfirmed || user.ID == "" || user.EstablishmentID == "" {
		t.Fatalf("user = %#v", user)
	}
}

func TestConfirmEmailAndCreateSessionRejectsEveryInvalidStateUniformly(t *testing.T) {
	tests := map[string]func(*verificationRecord){
		"missing":     nil,
		"expired":     func(record *verificationRecord) { record.ExpiresAt = registrationTestNow },
		"consumed":    func(record *verificationRecord) { record.Consumed = true },
		"invalidated": func(record *verificationRecord) { record.Invalidated = true },
		"confirmed":   func(record *verificationRecord) { record.EmailConfirmed = true },
		"inactive":    func(record *verificationRecord) { record.Active = false },
		"not owner":   func(record *verificationRecord) { record.Role = "professional" },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			transaction := validVerificationTransaction(t)
			if mutate == nil {
				transaction.found = false
			} else {
				mutate(&transaction.record)
			}
			err := testVerificationStore(transaction).ConfirmEmailAndCreateSession(context.Background(), nil, testSessionID, registrationTestNow, registrationTestNow, successfulIssue)
			if !errors.Is(err, ErrInvalidEmailConfirmation) || !reflect.DeepEqual(transaction.operations, []string{"lock"}) || !transaction.rolledBack {
				t.Fatalf("error = %v, operations = %#v, rollback = %t", err, transaction.operations, transaction.rolledBack)
			}
		})
	}
}

func TestConfirmEmailAndCreateSessionRejectsConcurrentReplay(t *testing.T) {
	for _, failed := range []string{"confirm-zero", "consume-zero"} {
		t.Run(failed, func(t *testing.T) {
			transaction := validVerificationTransaction(t)
			transaction.failAt = failed
			err := testVerificationStore(transaction).ConfirmEmailAndCreateSession(context.Background(), nil, testSessionID, registrationTestNow, registrationTestNow, successfulIssue)
			if !errors.Is(err, ErrInvalidEmailConfirmation) || !transaction.rolledBack {
				t.Fatalf("error = %v, rollback = %t", err, transaction.rolledBack)
			}
		})
	}
}

func TestConfirmEmailAndCreateSessionRollsBackTransactionalFailures(t *testing.T) {
	for _, failed := range []string{"lock", "confirm", "consume", "trial", "session", "commit"} {
		t.Run(failed, func(t *testing.T) {
			transaction := validVerificationTransaction(t)
			transaction.failAt = failed
			err := testVerificationStore(transaction).ConfirmEmailAndCreateSession(context.Background(), nil, testSessionID, registrationTestNow, registrationTestNow, successfulIssue)
			if err == nil || !transaction.rolledBack {
				t.Fatalf("error = %v, rollback = %t", err, transaction.rolledBack)
			}
		})
	}
}

func TestConfirmEmailAndCreateSessionRollsBackWhenAuthenticationIssuanceFails(t *testing.T) {
	transaction := validVerificationTransaction(t)
	issueFailure := errors.New("authentication issuance failed")
	err := testVerificationStore(transaction).ConfirmEmailAndCreateSession(
		context.Background(), nil, testSessionID, registrationTestNow, registrationTestNow,
		func(auth.User) error { return issueFailure },
	)
	if !errors.Is(err, issueFailure) || !transaction.rolledBack {
		t.Fatalf("error = %v, operations = %#v, rollback = %t", err, transaction.operations, transaction.rolledBack)
	}
	if reflect.DeepEqual(transaction.operations, []string{"lock", "confirm", "consume", "trial", "session", "commit"}) {
		t.Fatalf("transaction committed after issuance failure: %#v", transaction.operations)
	}
}

func successfulIssue(auth.User) error { return nil }

func TestVerificationWriteErrorTreatsExistingTrialAsInvalidConfirmation(t *testing.T) {
	for _, constraint := range []string{"establishment_trials_owner_unique", "establishment_trials_establishment_unique"} {
		err := verificationWriteError("create establishment trial", &pgconn.PgError{ConstraintName: constraint})
		if !errors.Is(err, ErrInvalidEmailConfirmation) {
			t.Fatalf("constraint %s returned %v", constraint, err)
		}
	}
}

func testVerificationStore(transaction verificationTransaction) *Store {
	return &Store{beginVerification: func(context.Context) (verificationTransaction, error) { return transaction, nil }}
}

func validVerificationTransaction(t *testing.T) *verificationTransactionStub {
	t.Helper()
	var userID, establishmentID, confirmationID pgtype.UUID
	_ = userID.Scan("018f47d2-e886-7f1c-8e8d-8b38eaf16463")
	_ = establishmentID.Scan("018f47d2-e886-7f1c-8e8d-8b38eaf16464")
	_ = confirmationID.Scan("018f47d2-e886-7f1c-8e8d-8b38eaf16465")
	return &verificationTransactionStub{found: true, record: verificationRecord{
		ConfirmationID: confirmationID, UserID: userID, EstablishmentID: establishmentID, Email: "owner@example.com",
		Role: "owner", Active: true, CredentialVersion: 1, ExpiresAt: registrationTestNow.Add(time.Nanosecond),
	}}
}

type verificationTransactionStub struct {
	record     verificationRecord
	found      bool
	failAt     string
	operations []string
	rolledBack bool
	trial      TrialWindow
}

func (stub *verificationTransactionStub) operation(name string) error {
	stub.operations = append(stub.operations, name)
	if stub.failAt == name {
		return errors.New("transaction operation failed")
	}
	return nil
}
func (stub *verificationTransactionStub) Lock(context.Context, []byte) (verificationRecord, bool, error) {
	return stub.record, stub.found, stub.operation("lock")
}
func (stub *verificationTransactionStub) ConfirmUser(context.Context, pgtype.UUID, time.Time) (bool, error) {
	err := stub.operation("confirm")
	return stub.failAt != "confirm-zero", err
}
func (stub *verificationTransactionStub) Consume(context.Context, pgtype.UUID, time.Time) (bool, error) {
	err := stub.operation("consume")
	return stub.failAt != "consume-zero", err
}
func (stub *verificationTransactionStub) CreateTrial(_ context.Context, _ verificationRecord, trial TrialWindow) error {
	stub.trial = trial
	return stub.operation("trial")
}
func (stub *verificationTransactionStub) CreateSession(context.Context, pgtype.UUID, pgtype.UUID, time.Time) error {
	return stub.operation("session")
}
func (stub *verificationTransactionStub) Commit(context.Context) error {
	return stub.operation("commit")
}
func (stub *verificationTransactionStub) Rollback(context.Context) error {
	stub.rolledBack = true
	return nil
}
