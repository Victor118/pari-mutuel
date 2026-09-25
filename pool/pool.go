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
)

type Pool struct {
	ID              PoolID
	Creator         AccountID
	Resolver        ResolverID
	Question        Question
	Outcomes        []OutcomeID
	ClosesAt        time.Time
	stakedByOutcome map[OutcomeID]Amount
	winner          OutcomeID
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
		ID:              poolID,
		Creator:         creator,
		Resolver:        resolver,
		Question:        question,
		Outcomes:        slices.Clone(outcomes),
		ClosesAt:        closesAt,
		stakedByOutcome: stakedByOutcome,
	}

	return pool, nil
}

func (p *Pool) PlaceBet(account AccountID, outcome OutcomeID, amount Amount, now time.Time) error {
	if now.After(p.ClosesAt) {
		return fmt.Errorf("%w : pool is closed since %v", ErrClosedPool, p.ClosesAt)
	}
	if _, resolved := p.Winner(); resolved {
		return fmt.Errorf("%w : pool %v resolved on %v", ErrAlreadyResolved, p.ID, p.winner)
	}

	if !slices.Contains(p.Outcomes, outcome) {
		return fmt.Errorf("%w : outcome %v not exist for the pool %v", ErrUnknownOutcome, outcome, p.ID)
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
	if oracle != p.Resolver {
		return fmt.Errorf("%w : got %v should be %v", ErrNotResolver, oracle, p.Resolver)
	}
	if _, resolved := p.Winner(); resolved {
		return fmt.Errorf("%w : pool %v resolved on %v", ErrAlreadyResolved, p.ID, p.winner)
	}
	if !slices.Contains(p.Outcomes, winner) {
		return fmt.Errorf("%w : outcome %v not exist for the pool %v", ErrUnknownOutcome, winner, p.ID)
	}
	p.winner = winner
	return nil
}

func (p *Pool) Winner() (OutcomeID, bool) {
	return p.winner, p.winner != ""
}
