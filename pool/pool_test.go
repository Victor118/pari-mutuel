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

func mustPlaceBet(t *testing.T, p *pool.Pool, account pool.AccountID, outcome pool.OutcomeID, stake pool.Money) {
	t.Helper()
	if err := p.PlaceBet(account, outcome, stake, time.Now()); err != nil {
		t.Fatalf("setup: bet refused : %v", err)
	}
}

const (
	testCurrency    = pool.Currency("USD")
	foreignCurrency = pool.Currency("EUR")
)

func testMoney(t *testing.T, cents int64) pool.Money {
	t.Helper()
	return pool.NewMoney(mustAmount(t, cents), testCurrency)
}

func testMoneyIn(t *testing.T, cents int64, currency pool.Currency) pool.Money {
	t.Helper()
	return pool.NewMoney(mustAmount(t, cents), currency)
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
		testCurrency,
	)
	if err != nil {
		t.Fatalf("setup: pool refused : %v", err)
	}
	return p
}

func TestNewPool_LessThanTwoOutcomes_ReturnsError(t *testing.T) {
	closesAt := time.Now().AddDate(0, 1, 0)
	p, err := pool.NewPool(pool.PoolID("p1"), pool.AccountID("alice"), pool.ResolverID("oracle1"), mustQuestion(t, "PSG-OM ?"), []pool.OutcomeID{}, closesAt, testCurrency)
	if !errors.Is(err, pool.ErrNotEnoughOutcomes) {
		t.Errorf("err = %v, want ErrNotEnoughOutcomes", err)
	}
	if p != nil {
		t.Errorf("pool should not be created with less than two outcomes")
	}

}

func TestNewPool_DuplicateOutcomes_ReturnsError(t *testing.T) {
	closesAt := time.Now().AddDate(0, 1, 0)
	p, err := pool.NewPool(pool.PoolID("p1"), pool.AccountID("alice"), pool.ResolverID("oracle1"), mustQuestion(t, "PSG-OM ?"), []pool.OutcomeID{"psg", "om", "psg"}, closesAt, testCurrency)
	if !errors.Is(err, pool.ErrDuplicateOutcome) {
		t.Errorf("err = %v, want ErrDuplicateOutcome", err)
	}
	if p != nil {
		t.Errorf("pool should not be created with duplicate outcomes")
	}
}

func TestPlaceBet_StoresStake(t *testing.T) {
	//Given
	p := newTestPool(t, time.Now().AddDate(0, 1, 0), "psg", "om")
	outcome := pool.OutcomeID("psg")
	stake := testMoney(t, 12000)

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
	stake := testMoney(t, 12000)
	if err := p.PlaceBet(pool.AccountID("bob"), outcome, stake, time.Now()); !errors.Is(err, pool.ErrUnknownOutcome) {
		t.Errorf("err = %v, want ErrUnknownOutcome", err)
	}
}

func TestPlaceBet_DifferentOutcomes_TrackedSeparately(t *testing.T) {
	p := newTestPool(t, time.Now().AddDate(0, 1, 0), "psg", "om")
	outcomePsg := pool.OutcomeID("psg")
	amountPsg := testMoney(t, 10000)
	outcomeOm := pool.OutcomeID("om")
	amountOm := testMoney(t, 5000)
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
	first := testMoney(t, 12000)
	second := testMoney(t, 8000)
	want := testMoney(t, 20000)

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
	amount := testMoney(t, 10000)
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
	amount := testMoney(t, 10000)
	if err := p.PlaceBet(pool.AccountID("alice"), winner, amount, time.Now()); !errors.Is(err, pool.ErrAlreadyResolved) {
		t.Errorf("err : %v, want ErrAlreadyResolved", err)
	}

	totalStaked := p.TotalBetOnOutcome(winner)

	if totalStaked != pool.Zero(testCurrency) {
		t.Errorf("bet should be refused, total amount should be zero, got %v", totalStaked)
	}
}

func TestCancel_ByDesignatedResolver_MarksCancelled(t *testing.T) {
	p := newTestPool(t, time.Now().AddDate(0, 0, 1), "psg", "om", "nul")
	err := p.Cancel(pool.ResolverID("oracle1"))
	if err != nil {
		t.Errorf("cancel refused : %v", err)
	}
	if !p.IsCancelled() {
		t.Errorf("should be cancelled")
	}

}

func TestCancel_ByUnknownResolver_NotAccepted(t *testing.T) {
	p := newTestPool(t, time.Now().AddDate(0, 0, 1), "psg", "om", "nul")
	err := p.Cancel(pool.ResolverID("unknown"))
	if !errors.Is(err, pool.ErrNotResolver) {
		t.Errorf("err : %v, want ErrNotResolver", err)
	}
	if p.IsCancelled() {
		t.Errorf("should not be cancelled")
	}
}

func TestNewPool_NotCancelledAtCreation(t *testing.T) {
	p := newTestPool(t, time.Now().AddDate(0, 0, 1), "psg", "om", "nul")
	if p.IsCancelled() {
		t.Errorf("pool should not be cancelled at creation")
	}

}

func TestPlaceBet_OnCancelledPool_NotAccepted(t *testing.T) {
	p := newTestPool(t, time.Now().AddDate(0, 0, 1), "psg", "om", "nul")
	_ = p.Cancel(pool.ResolverID("oracle1"))
	outcome := pool.OutcomeID("psg")
	amount := testMoney(t, 12000)
	if err := p.PlaceBet(pool.AccountID("alice"), outcome, amount, time.Now()); !errors.Is(err, pool.ErrPoolCancelled) {
		t.Errorf("want ErrPoolCancelled, got %v", err)
	}
	if p.TotalBetOnOutcome(outcome) != pool.Zero(testCurrency) {
		t.Errorf("amount should be zero")
	}
}

func TestCancel_OnResolvedPool_ReturnsError(t *testing.T) {
	p := newTestPool(t, time.Now().AddDate(0, 0, 1), "psg", "om", "nul")
	winner := pool.OutcomeID("om")
	err := p.Resolve(pool.ResolverID("oracle1"), winner)
	if err != nil {
		t.Fatalf("setup error %v", err)
	}

	if err := p.Cancel(pool.ResolverID("oracle1")); !errors.Is(err, pool.ErrAlreadyResolved) {
		t.Errorf("want ErrAlreadyResolved, got %v", err)
	}
	if p.IsCancelled() {
		t.Error("a resolved pool must not become cancelled")
	}
}

func TestResolve_OnCancelledPool_ReturnsError(t *testing.T) {
	p := newTestPool(t, time.Now().AddDate(0, 0, 1), "psg", "om", "nul")
	_ = p.Cancel(pool.ResolverID("oracle1"))
	winner := pool.OutcomeID("om")
	if err := p.Resolve(pool.ResolverID("oracle1"), winner); !errors.Is(err, pool.ErrPoolCancelled) {
		t.Errorf("want ErrPoolCancelled, got %v", err)
	}
	if _, resolved := p.Winner(); resolved {
		t.Error("a cancelled pool must not become resolved")
	}

}

func TestPlaceBet_WrongCurrency_NotAccepted(t *testing.T) {
	p := newTestPool(t, time.Now().AddDate(0, 0, 1), "psg", "om", "nul")
	outcome := pool.OutcomeID("om")
	foreign := testMoneyIn(t, 12000, foreignCurrency)

	err := p.PlaceBet(pool.AccountID("alice"), outcome, foreign, time.Now())

	if !errors.Is(err, pool.ErrCurrencyMismatch) {
		t.Errorf("err : %v, want ErrCurrencyMismatch", err)
	}
	if p.TotalBetOnOutcome(outcome) != pool.Zero(testCurrency) {
		t.Errorf("bet in foreign currency must not be recorded")
	}
}

func TestNewPool_EmptyCurrency_ReturnsError(t *testing.T) {
	p, err := pool.NewPool(
		pool.PoolID("p1"),
		pool.AccountID("alice"),
		pool.ResolverID("oracle1"),
		mustQuestion(t, "PSG-OM ?"),
		[]pool.OutcomeID{"psg", "om", "nul"},
		time.Now().AddDate(0, 0, 1),
		pool.Currency(""),
	)
	if !errors.Is(err, pool.ErrEmptyCurrency) {
		t.Errorf("got %v, want ErrEmptyCurrency", err)
	}
	if p != nil {
		t.Errorf("pool should not be created without currency")
	}

}

func TestNewPool_BlankCurrency_ReturnsError(t *testing.T) {
	p, err := pool.NewPool(
		pool.PoolID("p1"),
		pool.AccountID("alice"),
		pool.ResolverID("oracle1"),
		mustQuestion(t, "PSG-OM ?"),
		[]pool.OutcomeID{"psg", "om", "nul"},
		time.Now().AddDate(0, 0, 1),
		pool.Currency(" "),
	)
	if !errors.Is(err, pool.ErrEmptyCurrency) {
		t.Errorf("got %v, want ErrEmptyCurrency", err)
	}
	if p != nil {
		t.Errorf("pool should not be created without currency")
	}
}

type testBet struct {
	account pool.AccountID
	outcome pool.OutcomeID
	cents   int64
}

func TestPayoutFor_ReturnsProportionalGain(t *testing.T) {
	cases := []struct {
		name    string
		bets    []testBet
		account pool.AccountID
		want    int64
	}{
		{
			name:    "parts égales, pot de 40 000",
			bets:    []testBet{{"alice", "psg", 10000}, {"carol", "psg", 10000}, {"bob", "om", 20000}},
			account: "alice",
			want:    20000,
		},
		{
			name:    "côté perdant plus petit",
			bets:    []testBet{{"alice", "psg", 10000}, {"carol", "psg", 10000}, {"bob", "om", 10000}},
			account: "alice",
			want:    15000,
		},
		{
			name:    "mises différentes sur le gagnant",
			bets:    []testBet{{"alice", "psg", 10000}, {"carol", "psg", 30000}, {"bob", "om", 40000}},
			account: "carol",
			want:    60000,
		},
		{
			name:    "mises cumulées d'un même compte",
			bets:    []testBet{{"alice", "psg", 6000}, {"alice", "psg", 4000}, {"carol", "psg", 10000}, {"bob", "om", 20000}},
			account: "alice",
			want:    20000,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			//given
			p := newTestPool(t, time.Now().AddDate(0, 0, 1), "psg", "om")
			for _, b := range c.bets {
				mustPlaceBet(t, p, b.account, b.outcome, testMoney(t, b.cents))
			}
			if err := p.Resolve(pool.ResolverID("oracle1"), "psg"); err != nil {
				t.Fatalf("setup: resolution refused: %v", err)
			}

			//when
			got, err := p.PayoutFor(c.account)

			//then
			if err != nil {
				t.Fatalf("payout refused : %v", err)
			}
			if want := testMoney(t, c.want); got != want {
				t.Errorf("PayoutFor(%v) = %v, want %v", c.account, got, want)
			}
		})
	}
}

func TestPayoutFor_LoserClaims_ReturnsZero(t *testing.T) {
	p := newTestPool(t, time.Now().AddDate(0, 0, 1), "psg", "om")
	// Un perdant ne touche rien, que sa mise dépasse la masse gagnante (alice) ou non (dave).
	mustPlaceBet(t, p, "alice", "psg", testMoney(t, 50000))
	mustPlaceBet(t, p, "dave", "psg", testMoney(t, 5000))
	mustPlaceBet(t, p, "bob", "om", testMoney(t, 10000))
	if err := p.Resolve(pool.ResolverID("oracle1"), "om"); err != nil {
		t.Fatalf("setup: resolution refused : %v", err)
	}

	for _, loser := range []pool.AccountID{"alice", "dave"} {
		got, err := p.PayoutFor(loser)
		if err != nil {
			t.Errorf("PayoutFor(%v) refused : %v", loser, err)
			continue
		}
		if got != pool.Zero(testCurrency) {
			t.Errorf("PayoutFor(%v) = %v, want zero", loser, got)
		}
	}
}

func TestPayoutFor_NobodyBetOnWinner_ReturnsZero(t *testing.T) {
	p := newTestPool(t, time.Now().AddDate(0, 0, 1), "psg", "om")
	mustPlaceBet(t, p, "alice", "psg", testMoney(t, 10000))
	mustPlaceBet(t, p, "carol", "psg", testMoney(t, 10000))
	if err := p.Resolve(pool.ResolverID("oracle1"), "om"); err != nil {
		t.Fatalf("setup: resolution refused : %v", err)
	}
	got, err := p.PayoutFor("alice")
	if err != nil {
		t.Errorf("Should return zero without error, got : %v", err)
	}
	if got != pool.Zero(testCurrency) {
		t.Errorf("got :%v , want zero", got)
	}
}

func TestPayoutFor_LoserWithNonZeroStake_ReturnsZero(t *testing.T) {
	p := newTestPool(t, time.Now().AddDate(0, 0, 1), "psg", "om")
	mustPlaceBet(t, p, "alice", "psg", testMoney(t, 10000))
	mustPlaceBet(t, p, "bob", "om", testMoney(t, 10000))
	if err := p.Resolve(pool.ResolverID("oracle1"), "om"); err != nil {
		t.Fatalf("setup: resolution refused : %v", err)
	}

	got, err := p.PayoutFor("alice")

	if err != nil {
		t.Fatalf("payout refused : %v", err)
	}
	if got != pool.Zero(testCurrency) {
		t.Errorf("PayoutFor(alice) = %v, want zero", got)
	}
}

func TestPayoutFor_AccountWhoNeverBet_ReturnsZero(t *testing.T) {
	p := newTestPool(t, time.Now().AddDate(0, 0, 1), "psg", "om")
	mustPlaceBet(t, p, "alice", "psg", testMoney(t, 10000))
	mustPlaceBet(t, p, "bob", "om", testMoney(t, 10000))
	if err := p.Resolve(pool.ResolverID("oracle1"), "psg"); err != nil {
		t.Fatalf("setup: resolution refused : %v", err)
	}

	got, err := p.PayoutFor("carol")

	if err != nil {
		t.Fatalf("payout refused : %v", err)
	}
	if got != pool.Zero(testCurrency) {
		t.Errorf("PayoutFor(carol) = %v, want zero", got)
	}
}

func TestPayoutFor_AccountWithStakesOnSeveralOutcomes_PaidOnWinningOnly(t *testing.T) {
	p := newTestPool(t, time.Now().AddDate(0, 0, 1), "psg", "om")
	mustPlaceBet(t, p, "alice", "psg", testMoney(t, 10000))
	mustPlaceBet(t, p, "alice", "om", testMoney(t, 10000))
	mustPlaceBet(t, p, "bob", "om", testMoney(t, 20000))
	if err := p.Resolve(pool.ResolverID("oracle1"), "psg"); err != nil {
		t.Fatalf("setup: resolution refused : %v", err)
	}

	got, err := p.PayoutFor("alice")

	if err != nil {
		t.Fatalf("payout refused : %v", err)
	}
	// seule la mise sur psg compte : 10 000 × 40 000 / 10 000
	if want := testMoney(t, 40000); got != want {
		t.Errorf("PayoutFor(alice) = %v, want %v", got, want)
	}
}

func TestPlaceBet_EmptyAccount_ReturnsErrAccountRequired(t *testing.T) {
	cases := []struct {
		name    string
		account pool.AccountID
	}{
		{"vide", ""},
		{"blanc", "   "},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := newTestPool(t, time.Now().AddDate(0, 0, 1), "psg", "om")

			err := p.PlaceBet(c.account, "psg", testMoney(t, 10000), time.Now())

			if !errors.Is(err, pool.ErrAccountRequired) {
				t.Errorf("err = %v, want ErrAccountRequired", err)
			}
			if staked := p.TotalBetOnOutcome("psg"); staked != pool.Zero(testCurrency) {
				t.Errorf("bet without account must not be recorded, got %v", staked)
			}
		})
	}
}

func TestPayoutFor_UnresolvedPool_ReturnsErrNotResolved(t *testing.T) {
	p := newTestPool(t, time.Now().AddDate(0, 0, 1), "psg", "om")
	mustPlaceBet(t, p, "alice", "psg", testMoney(t, 10000))

	got, err := p.PayoutFor("alice")

	if !errors.Is(err, pool.ErrNotResolved) {
		t.Errorf("err = %v, want ErrNotResolved", err)
	}
	if got != pool.Zero(testCurrency) {
		t.Errorf("PayoutFor(alice) = %v, want zero", got)
	}
}

func TestPayoutFor_CancelledPool_ReturnsErrPoolCancelled(t *testing.T) {
	p := newTestPool(t, time.Now().AddDate(0, 0, 1), "psg", "om")
	mustPlaceBet(t, p, "alice", "psg", testMoney(t, 10000))
	if err := p.Cancel(pool.ResolverID("oracle1")); err != nil {
		t.Fatalf("setup: cancel refused : %v", err)
	}

	got, err := p.PayoutFor("alice")

	if !errors.Is(err, pool.ErrPoolCancelled) {
		t.Errorf("err = %v, want ErrPoolCancelled", err)
	}
	if got != pool.Zero(testCurrency) {
		t.Errorf("PayoutFor(alice) = %v, want zero", got)
	}
}

func TestNewPool_EmptyResolver_ShouldFail(t *testing.T) {
	closesAt := time.Now().AddDate(0, 1, 0)
	_, err := pool.NewPool(
		pool.PoolID("p1"),
		pool.AccountID("alice"),
		pool.ResolverID(""),
		mustQuestion(t, "PSG-OM ?"),
		[]pool.OutcomeID{"om", "psg", "nul"},
		closesAt,
		testCurrency,
	)

	if !errors.Is(err, pool.ErrResolverRequired) {
		t.Errorf("got %v, want ErrResolverRequired", err)
	}

}

func TestNewPool_SpaceResolver_ShouldFail(t *testing.T) {
	closesAt := time.Now().AddDate(0, 1, 0)
	_, err := pool.NewPool(
		pool.PoolID("p1"),
		pool.AccountID("alice"),
		pool.ResolverID("  "),
		mustQuestion(t, "PSG-OM ?"),
		[]pool.OutcomeID{"om", "psg", "nul"},
		closesAt,
		testCurrency,
	)

	if !errors.Is(err, pool.ErrResolverRequired) {
		t.Errorf("got %v, want ErrResolverRequired", err)
	}

}

func TestPlaceBet_AmountZero_ShouldFail(t *testing.T) {

	p := newTestPool(t, time.Now().AddDate(0, 0, 1), "psg", "om")
	if err := p.PlaceBet(pool.AccountID("alice"), pool.OutcomeID("om"), testMoney(t, 0), time.Now()); !errors.Is(err, pool.ErrBetAmount) {
		t.Errorf("got %v,want ErrBetAmount", err)
	}
	staked := p.TotalBetOnOutcome(pool.OutcomeID("om"))
	if staked != pool.Zero(testCurrency) {
		t.Errorf("staked amount should be zero, got : %v", staked)
	}
}
