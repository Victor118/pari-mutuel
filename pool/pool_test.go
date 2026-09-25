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

func mustAmount(t *testing.T, cents int64) pool.Amount {
	t.Helper()
	a, err := pool.NewAmount(cents)
	if err != nil {
		t.Fatalf("setup: amount refused : %v", err)
	}
	return a
}

func newTestPool(t *testing.T, closesAt time.Time, outcomes ...pool.OutcomeID) *pool.Pool {
	t.Helper()
	p, err := pool.NewPool(
		pool.PoolID("p1"),
		pool.AccountID("alice"),
		pool.ResolverID("oracle1"),
		mustQuestion(t, "PSG-OM ?"),
		outcomes,
		closesAt,
	)
	if err != nil {
		t.Fatalf("setup: pool refused : %v", err)
	}
	return p
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
	p := newTestPool(t, time.Now().AddDate(0, 1, 0), "psg", "om")
	outcome := pool.OutcomeID("psg")
	stake := mustAmount(t, 12000)

	//When
	if err := p.PlaceBet(pool.AccountID("bob"), outcome, stake, time.Now()); err != nil {
		t.Fatalf("bet refused : %v", err)
	}

	//Then
	totalStaked := p.TotalBetOnOutcome(outcome)

	if totalStaked != stake {
		t.Errorf("TotalBetOnOutcome(%v) = %v, want %v", outcome, totalStaked, stake)
	}
}

func TestPlaceBet_UnknownOutcome_ReturnsError(t *testing.T) {
	p := newTestPool(t, time.Now().AddDate(0, 1, 0), "psg", "om")
	outcome := pool.OutcomeID("lyon")
	stake := mustAmount(t, 12000)
	if err := p.PlaceBet(pool.AccountID("bob"), outcome, stake, time.Now()); !errors.Is(err, pool.ErrUnknownOutcome) {
		t.Errorf("err = %v, want ErrUnknownOutcome", err)
	}
}

func TestPlaceBet_DifferentOutcomes_TrackedSeparately(t *testing.T) {
	p := newTestPool(t, time.Now().AddDate(0, 1, 0), "psg", "om")
	outcomePsg := pool.OutcomeID("psg")
	amountPsg := mustAmount(t, 10000)
	outcomeOm := pool.OutcomeID("om")
	amountOm := mustAmount(t, 5000)
	if err := p.PlaceBet(pool.AccountID("alice"), outcomePsg, amountPsg, time.Now()); err != nil {
		t.Fatalf("bet refused : %v", err)
	}
	if err := p.PlaceBet(pool.AccountID("bob"), outcomeOm, amountOm, time.Now()); err != nil {
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

func TestPlaceBet_SameOutcomeTwice_AmountsAccumulate(t *testing.T) {
	p := newTestPool(t, time.Now().AddDate(0, 1, 0), "psg", "om")
	outcome := pool.OutcomeID("psg")
	first := mustAmount(t, 12000)
	second := mustAmount(t, 8000)
	want := mustAmount(t, 20000)

	if err := p.PlaceBet(pool.AccountID("alice"), outcome, first, time.Now()); err != nil {
		t.Fatalf("bet refused : %v", err)
	}
	if err := p.PlaceBet(pool.AccountID("bob"), outcome, second, time.Now()); err != nil {
		t.Fatalf("bet refused : %v", err)
	}

	totalStaked := p.TotalBetOnOutcome(outcome)
	if totalStaked != want {
		t.Errorf("TotalBetOnOutcome(%v) = %v, want %v", outcome, totalStaked, want)
	}
}

func TestPlaceBet_AfterClosesAt_NotAccepted(t *testing.T) {
	p := newTestPool(t, time.Now().AddDate(0, -1, 0), "psg", "om")
	outcome := pool.OutcomeID("psg")
	amount := mustAmount(t, 10000)
	if err := p.PlaceBet(pool.AccountID("alice"), outcome, amount, time.Now()); !errors.Is(err, pool.ErrClosedPool) {
		t.Errorf("bet after closesAt should not be accepted")
	}
}

func TestResolve_ByDesignatedResolver_RecordsWinner(t *testing.T) {
	p := newTestPool(t, time.Now().AddDate(0, 0, 1), "psg", "om", "nul")
	winner := pool.OutcomeID("om")
	err := p.Resolve(pool.ResolverID("oracle1"), pool.OutcomeID("om"))
	if err != nil {
		t.Fatalf("resolution refused : %v", err)
	}
	got, resolved := p.Winner()
	if !resolved {
		t.Fatalf("pool not resolved after Resolve")
	}
	if winner != got {
		t.Errorf("Winner() = %v, want %v", got, winner)
	}
}

func TestResolve_ByOtherThanResolver_ReturnsError(t *testing.T) {
	p := newTestPool(t, time.Now().AddDate(0, 0, 1), "psg", "om", "nul")
	winner := pool.OutcomeID("om")
	err := p.Resolve(pool.ResolverID("other"), winner)
	if !errors.Is(err, pool.ErrNotResolver) {
		t.Errorf("err : %v, want ErrNotResolver", err)
	}
	_, resolved := p.Winner()
	if resolved {
		t.Errorf("pool resolved by un unauthorized resolver")
	}
}

func TestResolve_AlreadyResolvedPool_ReturnsError(t *testing.T) {
	p := newTestPool(t, time.Now().AddDate(0, 0, 1), "psg", "om", "nul")
	winner := pool.OutcomeID("om")
	err := p.Resolve(pool.ResolverID("oracle1"), winner)
	if err != nil {
		t.Fatalf("resolve should not fail : %v", err)
	}
	newWinner := pool.OutcomeID("psg")

	if err := p.Resolve(pool.ResolverID("oracle1"), newWinner); !errors.Is(err, pool.ErrAlreadyResolved) {
		t.Errorf("err : %v, want ErrAlreadyResolved", err)
	}

	got, _ := p.Winner()
	if got != winner {
		t.Errorf("winner should be : %v, got %v", winner, got)
	}
}

func TestResolve_UnknownOutcome_ReturnsError(t *testing.T) {
	p := newTestPool(t, time.Now().AddDate(0, 0, 1), "psg", "om", "nul")
	winner := pool.OutcomeID("lyon")

	if err := p.Resolve(pool.ResolverID("oracle1"), winner); !errors.Is(err, pool.ErrUnknownOutcome) {
		t.Errorf("err : %v, want ErrUnknownOutcome", err)
	}

	if _, resolved := p.Winner(); resolved {
		t.Errorf("pool resolved on an outcome that does not exist")
	}
}

func TestPlaceBet_OnResolvedPool_NotAccepted(t *testing.T) {
	p := newTestPool(t, time.Now().AddDate(0, 0, 1), "psg", "om", "nul")
	winner := pool.OutcomeID("om")
	err := p.Resolve(pool.ResolverID("oracle1"), winner)
	if err != nil {
		t.Fatalf("resolve should not fail : %v", err)
	}
	amount := mustAmount(t, 10000)
	if err := p.PlaceBet(pool.AccountID("alice"), winner, amount, time.Now()); !errors.Is(err, pool.ErrAlreadyResolved) {
		t.Errorf("err : %v, want ErrAlreadyResolved", err)
	}

	totalStaked := p.TotalBetOnOutcome(winner)

	if totalStaked != pool.Zero() {
		t.Errorf("bet should be refused, total amount should be zero, got %v", totalStaked)
	}
}
