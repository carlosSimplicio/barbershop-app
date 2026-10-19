package onboarding

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestReplaceEmailConfirmationLocksInvalidatesCreatesAndCommits(t *testing.T) {
	transaction := &confirmationTransactionStub{eligible: true}
	store := testConfirmationStore(transaction)

	confirmation, replaced, err := store.ReplaceEmailConfirmation(context.Background(), "018f47d2-e886-7f1c-8e8d-8b38eaf16463")
	if err != nil || !replaced {
		t.Fatalf("confirmation = %#v, replaced = %t, error = %v", confirmation, replaced, err)
	}
	if !reflect.DeepEqual(transaction.operations, []string{"lock", "invalidate", "create", "commit"}) {
		t.Fatalf("operations = %#v", transaction.operations)
	}
	if !transaction.invalidatedAt.Equal(registrationTestNow) || !transaction.createdAt.Equal(registrationTestNow) {
		t.Fatalf("timestamps = invalidate %s, create %s", transaction.invalidatedAt, transaction.createdAt)
	}
	if !transaction.confirmation.ExpiresAt.Equal(registrationTestNow.Add(24*time.Hour)) || len(transaction.confirmation.TokenHash) != 32 {
		t.Fatalf("created confirmation = %#v", transaction.confirmation)
	}
}

func TestReplaceEmailConfirmationStopsWhenOwnerBecameIneligible(t *testing.T) {
	transaction := &confirmationTransactionStub{eligible: false}
	store := testConfirmationStore(transaction)

	_, replaced, err := store.ReplaceEmailConfirmation(context.Background(), "018f47d2-e886-7f1c-8e8d-8b38eaf16463")
	if err != nil || replaced {
		t.Fatalf("replaced = %t, error = %v", replaced, err)
	}
	if !reflect.DeepEqual(transaction.operations, []string{"lock"}) || !transaction.rolledBack {
		t.Fatalf("operations = %#v, rolledBack = %t", transaction.operations, transaction.rolledBack)
	}
}

func TestReplaceEmailConfirmationRollsBackEveryTransactionalFailure(t *testing.T) {
	for _, failedOperation := range []string{"lock", "invalidate", "create", "commit"} {
		t.Run(failedOperation, func(t *testing.T) {
			transaction := &confirmationTransactionStub{eligible: true, failAt: failedOperation}
			store := testConfirmationStore(transaction)

			_, replaced, err := store.ReplaceEmailConfirmation(context.Background(), "018f47d2-e886-7f1c-8e8d-8b38eaf16463")
			if err == nil || replaced || !transaction.rolledBack {
				t.Fatalf("replaced = %t, error = %v, rolledBack = %t", replaced, err, transaction.rolledBack)
			}
		})
	}
}

func testConfirmationStore(transaction confirmationTransaction) *Store {
	return &Store{
		now: func() time.Time { return registrationTestNow },
		beginConfirmation: func(context.Context) (confirmationTransaction, error) {
			return transaction, nil
		},
	}
}

type confirmationTransactionStub struct {
	eligible      bool
	failAt        string
	operations    []string
	rolledBack    bool
	invalidatedAt time.Time
	createdAt     time.Time
	confirmation  EmailConfirmation
}

func (stub *confirmationTransactionStub) operation(name string) error {
	stub.operations = append(stub.operations, name)
	if stub.failAt == name {
		return errors.New("transaction operation failed")
	}
	return nil
}

func (stub *confirmationTransactionStub) LockEligibleOwner(context.Context, pgtype.UUID) (bool, error) {
	if err := stub.operation("lock"); err != nil {
		return false, err
	}
	return stub.eligible, nil
}

func (stub *confirmationTransactionStub) InvalidateActive(_ context.Context, _ pgtype.UUID, at time.Time) error {
	stub.invalidatedAt = at
	return stub.operation("invalidate")
}

func (stub *confirmationTransactionStub) Create(_ context.Context, _ pgtype.UUID, confirmation EmailConfirmation, createdAt time.Time) error {
	stub.confirmation = confirmation
	stub.createdAt = createdAt
	return stub.operation("create")
}

func (stub *confirmationTransactionStub) Commit(context.Context) error {
	return stub.operation("commit")
}

func (stub *confirmationTransactionStub) Rollback(context.Context) error {
	stub.rolledBack = true
	return nil
}
