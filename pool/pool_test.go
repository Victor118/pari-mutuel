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

func TestNewPool_ErrorWhenLessThanTwoOutcomes(t *testing.T) {
	closesAt := time.Now().AddDate(0, 1, 0)
	_, err := pool.NewPool(pool.PoolID("p1"), pool.AccountID("alice"), pool.ResolverID("oracle1"), mustQuestion(t, "PSG-OM ?"), []pool.OutcomeID{}, closesAt)
	if !errors.Is(err, pool.ErrNotEnoughOutcomes) {
		t.Errorf("err = %v, want ErrNotEnoughOutcomes", err)
	}

}

func TestNewPool_ErrorWhenDuplicateOutcomes(t *testing.T) {
	closesAt := time.Now().AddDate(0, 1, 0)
	_, err := pool.NewPool(pool.PoolID("p1"), pool.AccountID("alice"), pool.ResolverID("oracle1"), mustQuestion(t, "PSG-OM ?"), []pool.OutcomeID{"psg", "om", "psg"}, closesAt)
	if !errors.Is(err, pool.ErrDuplicateOutcome) {
		t.Errorf("err = %v, want ErrDuplicateOutcome", err)
	}
}

func TestNewQuestion_ErrorWhenEmptyQuestion(t *testing.T) {

	_, err := pool.NewQuestion("")
	if !errors.Is(err, pool.ErrEmptyQuestion) {
		t.Errorf("err = %v, want ErrEmptyQuestion", err)
	}

}

func TestNewQuestion_Space_ErrorWhenEmptyQuestion(t *testing.T) {

	_, err := pool.NewQuestion(" ")
	if !errors.Is(err, pool.ErrEmptyQuestion) {
		t.Errorf("err = %v, want ErrEmptyQuestion", err)
	}

}

func TestPlaceNewBet_PoolIncrease(t *testing.T) {
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
	totalStaked, err := p.TotalBetOnOutcome(outcome)
	if err != nil {
		t.Errorf("total staked for outcom %v should exist ", outcome)
	}
	if totalStaked != stake {
		t.Errorf("TotalBetOnOutcome(%v) = %v, want %v", outcome, totalStaked, stake)
	}
}

func TestPlaceNewBet_BadOutcome_fail(t *testing.T) {
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
