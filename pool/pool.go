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

type PoolState string

const (
	Open      PoolState = "open"
	Settled   PoolState = "settled"
	Cancelled PoolState = "cancelled"
)

var (
	ErrNotEnoughOutcomes = errors.New("pool must have at least 2 outcomes")
	ErrDuplicateOutcome  = errors.New("outcomes must be unique")
	ErrEmptyQuestion     = errors.New("question cannot be empty")
	ErrUnknownOutcome    = errors.New("unknown outcome")
	ErrInvalidAmount     = errors.New("amount should be greater or equal to 0")
	ErrClosedPool        = errors.New("pool is closed")
)

type Pool struct {
	ID              PoolID
	Creator         AccountID
	Resolver        ResolverID
	Question        Question
	Outcomes        []OutcomeID
	State           PoolState
	ClosesAt        time.Time
	stakedByOutcome map[OutcomeID]Amount
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
		State:           Open,
		ClosesAt:        closesAt,
		stakedByOutcome: stakedByOutcome,
	}

	return pool, nil
}

func (p *Pool) PlaceBet(account AccountID, outcome OutcomeID, amount Amount, now time.Time) error {
	if now.After(p.ClosesAt) {
		return fmt.Errorf("%w : pool is closed since %v", ErrClosedPool, p.ClosesAt)
	}
	if !slices.Contains(p.Outcomes, outcome) {
		return fmt.Errorf("%w : outcome %v not exist for the pool %v", ErrUnknownOutcome, outcome, p.ID)
	}
	stakedAmount := p.stakedByOutcome[outcome]
	total, err := stakedAmount.Add(amount)
	if err != nil {
		return fmt.Errorf("%w : amount incorrect : %v", err, amount)
	}
	p.stakedByOutcome[outcome] = total
	return nil
}

func (p *Pool) TotalBetOnOutcome(outcome OutcomeID) Amount {

	return p.stakedByOutcome[outcome]
}
