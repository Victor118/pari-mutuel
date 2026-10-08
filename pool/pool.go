package pool

import (
	"errors"
	"fmt"
	"math/bits"
	"slices"
	"strings"
	"time"
)

type AccountID string
type ResolverID string
type OutcomeID string
type PoolID string

var (
	ErrNotEnoughOutcomes       = errors.New("pool must have at least 2 outcomes")
	ErrDuplicateOutcome        = errors.New("outcomes must be unique")
	ErrEmptyQuestion           = errors.New("question cannot be empty")
	ErrUnknownOutcome          = errors.New("unknown outcome")
	ErrInvalidAmount           = errors.New("amount should be greater or equal to 0")
	ErrClosedPool              = errors.New("pool is closed")
	ErrResolverRequired        = errors.New("resolver is mandatory")
	ErrNotResolver             = errors.New("unauthorized resolver")
	ErrAlreadyResolved         = errors.New("already resolved")
	ErrPoolCancelled           = errors.New("pool cancelled")
	ErrCurrencyMismatch        = errors.New("currency mismatch")
	ErrEmptyCurrency           = errors.New("currency cannot be empty")
	ErrStakeExceedsWinningMass = errors.New("stake exceeds winning mass")
	ErrBetAmount               = errors.New("amount should be greater than zero")
	ErrAccountRequired         = errors.New("account is mandatory")
	ErrNotResolved             = errors.New("pool not resolved")
	ErrAlreadyClaimed          = errors.New("payout already claimed")
)

type Pool struct {
	id              PoolID
	creator         AccountID
	resolver        ResolverID
	question        Question
	outcomes        []OutcomeID
	closesAt        time.Time
	stakedByOutcome map[OutcomeID]Amount
	stakes          map[AccountID]map[OutcomeID]Amount
	claimed         map[AccountID]bool
	winner          OutcomeID
	cancelled       bool
	currency        Currency
}

func NewPool(poolID PoolID, creator AccountID, resolver ResolverID, question Question, outcomes []OutcomeID, closesAt time.Time, currency Currency) (*Pool, error) {
	if strings.TrimSpace(string(currency)) == "" {
		return nil, ErrEmptyCurrency
	}
	if strings.TrimSpace(string(resolver)) == "" {
		return nil, ErrResolverRequired
	}
	if len(outcomes) <= 1 {
		return nil, ErrNotEnoughOutcomes
	}
	seen := make(map[OutcomeID]bool, len(outcomes))
	for _, outcome := range outcomes {
		if seen[outcome] {
			return nil, ErrDuplicateOutcome
		}
		seen[outcome] = true
	}
	stakedByOutcome := make(map[OutcomeID]Amount, len(outcomes))

	pool := &Pool{
		id:              poolID,
		creator:         creator,
		resolver:        resolver,
		question:        question,
		outcomes:        slices.Clone(outcomes),
		closesAt:        closesAt,
		stakedByOutcome: stakedByOutcome,
		stakes:          make(map[AccountID]map[OutcomeID]Amount),
		claimed:         make(map[AccountID]bool),
		currency:        currency,
	}

	return pool, nil
}

func (p *Pool) PlaceBet(account AccountID, outcome OutcomeID, m Money, now time.Time) error {
	if strings.TrimSpace(string(account)) == "" {
		return ErrAccountRequired
	}
	if p.IsCancelled() {
		return ErrPoolCancelled
	}
	if m.amount == Zero(p.currency).amount {
		return fmt.Errorf("%w : amount %v", ErrBetAmount, m)
	}
	if p.currency != m.currency {
		return fmt.Errorf("%w : got %v, want %v", ErrCurrencyMismatch, m.currency, p.currency)
	}
	if now.After(p.closesAt) {
		return fmt.Errorf("%w : pool is closed since %v", ErrClosedPool, p.closesAt)
	}
	if _, resolved := p.Winner(); resolved {
		return fmt.Errorf("%w : pool %v resolved on %v", ErrAlreadyResolved, p.id, p.winner)
	}

	if !slices.Contains(p.outcomes, outcome) {
		return fmt.Errorf("%w : outcome %v not exist for the pool %v", ErrUnknownOutcome, outcome, p.id)
	}
	stakedAmount := p.stakedByOutcome[outcome]
	total := stakedAmount.Add(m.amount)

	p.stakedByOutcome[outcome] = total

	accountStakes, ok := p.stakes[account]
	if !ok {
		accountStakes = make(map[OutcomeID]Amount)
		p.stakes[account] = accountStakes
	}
	accountStakes[outcome] = accountStakes[outcome].Add(m.amount)
	return nil
}

func (p *Pool) TotalBetOnOutcome(outcome OutcomeID) Money {

	return NewMoney(p.stakedByOutcome[outcome], p.currency)
}

func (p *Pool) Resolve(oracle ResolverID, winner OutcomeID) error {
	if p.IsCancelled() {
		return fmt.Errorf("%w : can't resolve a cancelled pool", ErrPoolCancelled)
	}
	if oracle != p.resolver {
		return fmt.Errorf("%w : got %v should be %v", ErrNotResolver, oracle, p.resolver)
	}
	if _, resolved := p.Winner(); resolved {
		return fmt.Errorf("%w : pool %v resolved on %v", ErrAlreadyResolved, p.id, p.winner)
	}
	if !slices.Contains(p.outcomes, winner) {
		return fmt.Errorf("%w : outcome %v not exist for the pool %v", ErrUnknownOutcome, winner, p.id)
	}
	p.winner = winner
	return nil
}

func (p *Pool) Winner() (OutcomeID, bool) {
	return p.winner, p.winner != ""
}

func (p *Pool) IsCancelled() bool {
	return p.cancelled
}

func (p *Pool) Cancel(oracle ResolverID) error {
	if oracle != p.resolver {
		return fmt.Errorf("%w : want %v got %v", ErrNotResolver, p.resolver, oracle)
	}
	if _, resolved := p.Winner(); resolved {
		return fmt.Errorf("%w : can't cancelled a resolved pool", ErrAlreadyResolved)
	}
	p.cancelled = true
	return nil
}

func (p *Pool) PayoutFor(account AccountID) (Money, error) {
	if p.IsCancelled() {
		return Zero(p.currency), fmt.Errorf("%w : no payout on a cancelled pool", ErrPoolCancelled)
	}
	if _, resolved := p.Winner(); !resolved {
		return Zero(p.currency), fmt.Errorf("%w : pool %v", ErrNotResolved, p.id)
	}
	stake := p.stakes[account][p.winner]
	if stake.cents == 0 {
		return Zero(p.currency), nil
	}
	var total Amount
	for _, outcome := range p.outcomes {
		total = total.Add(p.stakedByOutcome[outcome])
	}
	winningMass := p.stakedByOutcome[p.winner]
	if winningMass.cents == 0 {
		return Zero(p.currency), nil
	}
	if stake.cents > winningMass.cents {
		return Zero(p.currency), fmt.Errorf("%w : stake %v, winning mass %v", ErrStakeExceedsWinningMass, stake, winningMass)
	}
	gain, err := NewAmount(mulDiv(stake.cents, total.cents, winningMass.cents))
	if err != nil {
		return Zero(p.currency), err
	}
	return NewMoney(gain, p.currency), nil
}

func (p *Pool) Claim(account AccountID) (Money, error) {
	if p.claimed[account] {
		return Zero(p.currency), fmt.Errorf("%w : account %v", ErrAlreadyClaimed, account)
	}
	payout, err := p.PayoutFor(account)
	if err != nil {
		return Zero(p.currency), err
	}
	p.claimed[account] = true
	return payout, nil
}

// mulDiv calcule a*b/c sans débordement. Préconditions : a,b >= 0, c > 0, a <= c.
func mulDiv(a, b, c int64) int64 {
	hi, lo := bits.Mul64(uint64(a), uint64(b))
	q, _ := bits.Div64(hi, lo, uint64(c))
	return int64(q)
}
