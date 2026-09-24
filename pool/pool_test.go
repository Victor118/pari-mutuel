package pool_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Victor118/pari-mutuel/pool"
)

func mustQuestion(t *testing.T, s string) pool.Question {
	t.Helper()
	q, err := pool.NewQuestion(s)
	if err != nil {
		t.Fatalf("setup: question refused : %v", err)
	}
	return q
}

func TestNewPool_OpenAtCreation(t *testing.T) {
	//Given
	//When
	closesAt := time.Now().AddDate(0, 1, 0)
	p, err := pool.NewPool(pool.PoolID("p1"), pool.AccountID("alice"), pool.ResolverID("oracle1"), mustQuestion(t, "PSG-OM ?"), []pool.OutcomeID{"psg", "om"}, closesAt)

	//Then
	if err != nil {
		t.Fatalf("Creation refused : %v", err)
	}
	if p.State != pool.Open {
		t.Errorf("state = %v, want Open", p.State)
	}
}

func TestNewPool_LessThanTwoOutcomes_ReturnsError(t *testing.T) {
	closesAt := time.Now().AddDate(0, 1, 0)
	_, err := pool.NewPool(pool.PoolID("p1"), pool.AccountID("alice"), pool.ResolverID("oracle1"), mustQuestion(t, "PSG-OM ?"), []pool.OutcomeID{}, closesAt)
	if !errors.Is(err, pool.ErrNotEnoughOutcomes) {
		t.Errorf("err = %v, want ErrNotEnoughOutcomes", err)
	}

}

func TestNewPool_DuplicateOutcomes_ReturnsError(t *testing.T) {
	closesAt := time.Now().AddDate(0, 1, 0)
	_, err := pool.NewPool(pool.PoolID("p1"), pool.AccountID("alice"), pool.ResolverID("oracle1"), mustQuestion(t, "PSG-OM ?"), []pool.OutcomeID{"psg", "om", "psg"}, closesAt)
	if !errors.Is(err, pool.ErrDuplicateOutcome) {
		t.Errorf("err = %v, want ErrDuplicateOutcome", err)
	}
}

func TestPlaceBet_StoresStake(t *testing.T) {
	//Given
	closesAt := time.Now().AddDate(0, 1, 0)
	p, err := pool.NewPool(pool.PoolID("p1"), pool.AccountID("alice"), pool.ResolverID("oracle1"), mustQuestion(t, "PSG-OM ?"), []pool.OutcomeID{"psg", "om"}, closesAt)
	if err != nil {
		t.Fatalf("unexpected error when create pool : %v", err)
	}
	outcome := pool.OutcomeID("psg")
	stake, _ := pool.NewAmount(12000)

	//When
	if err := p.PlaceBet(pool.AccountID("bob"), outcome, stake); err != nil {
		t.Fatalf("bet refused : %v", err)
	}

	//Then
	totalStaked := p.TotalBetOnOutcome(outcome)

	if totalStaked != stake {
		t.Errorf("TotalBetOnOutcome(%v) = %v, want %v", outcome, totalStaked, stake)
	}
}

func TestPlaceBet_UnknownOutcome_ReturnsError(t *testing.T) {
	closesAt := time.Now().AddDate(0, 1, 0)
	p, err := pool.NewPool(pool.PoolID("p1"), pool.AccountID("alice"), pool.ResolverID("oracle1"), mustQuestion(t, "PSG-OM ?"), []pool.OutcomeID{"psg", "om"}, closesAt)
	if err != nil {
		t.Fatalf("unexpected error when create pool : %v", err)
	}
	outcome := pool.OutcomeID("lyon")
	stake, _ := pool.NewAmount(12000)
	if err := p.PlaceBet(pool.AccountID("bob"), outcome, stake); !errors.Is(err, pool.ErrUnknownOutcome) {
		t.Errorf("err = %v, want ErrUnknownOutcome", err)
	}
}

func TestPlaceBet_DifferentOutcomes_TrackedSeparately(t *testing.T) {
	closesAt := time.Now().AddDate(0, 1, 0)
	p, err := pool.NewPool(pool.PoolID("p1"), pool.AccountID("alice"), pool.ResolverID("oracle1"), mustQuestion(t, "PSG-OM ?"), []pool.OutcomeID{"psg", "om"}, closesAt)
	if err != nil {
		t.Fatalf("unexpected error when create pool : %v", err)
	}
	outcomePsg := pool.OutcomeID("psg")
	amountPsg, _ := pool.NewAmount(10000)
	outcomeOm := pool.OutcomeID("om")
	amountOm, _ := pool.NewAmount(5000)
	if err := p.PlaceBet(pool.AccountID("alice"), outcomePsg, amountPsg); err != nil {
		t.Fatalf("bet refused : %v", err)
	}
	if err := p.PlaceBet(pool.AccountID("bob"), outcomeOm, amountOm); err != nil {
		t.Fatalf("bet refused : %v", err)
	}

	totalStakedPsg := p.TotalBetOnOutcome(outcomePsg)

	if totalStakedPsg != amountPsg {
		t.Errorf("TotalBetOnOutcome(%v) = %v, want %v", outcomePsg, totalStakedPsg, amountPsg)
	}

	totalStakedOm := p.TotalBetOnOutcome(outcomeOm)

	if totalStakedOm != amountOm {
		t.Errorf("TotalBetOnOutcome(%v) = %v, want %v", outcomeOm, totalStakedOm, amountOm)
	}
}
