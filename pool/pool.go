package pool

import (
	"errors"
	"fmt"
	"slices"
	"time"
)

type AccountID string
type ResolverID string
type OutcomeID string
type PoolID string

var (
	ErrNotEnoughOutcomes = errors.New("pool must have at least 2 outcomes")
	ErrDuplicateOutcome  = errors.New("outcomes must be unique")
	ErrEmptyQuestion     = errors.New("question cannot be empty")
	ErrUnknownOutcome    = errors.New("unknown outcome")
	ErrInvalidAmount     = errors.New("amount should be greater or equal to 0")
	ErrClosedPool        = errors.New("pool is closed")
	ErrNotResolver       = errors.New("unauthorized resolver")
	ErrAlreadyResolved   = errors.New("already resolved")
	ErrPoolCancelled     = errors.New("pool cancelled")
)

type Pool struct {
	id              PoolID
	creator         AccountID
	resolver        ResolverID
	question        Question
	outcomes        []OutcomeID
	closesAt        time.Time
	stakedByOutcome map[OutcomeID]Amount
	winner          OutcomeID
	cancelled       bool
}

func NewPool(poolID PoolID, creator AccountID, resolver ResolverID, question Question, outcomes []OutcomeID, closesAt time.Time) (*Pool, error) {
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
	}

	return pool, nil
}

func (p *Pool) PlaceBet(account AccountID, outcome OutcomeID, amount Amount, now time.Time) error {
	if p.IsCancelled() {
		return ErrPoolCancelled
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
	total := stakedAmount.Add(amount)

	p.stakedByOutcome[outcome] = total
	return nil
}

func (p *Pool) TotalBetOnOutcome(outcome OutcomeID) Amount {

	return p.stakedByOutcome[outcome]
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
