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
)

type Pool struct {
	ID              PoolID
	Creator         AccountID
	Resolver        ResolverID
	Question        Question
	Outcomes        []OutcomeID
	State           PoolState
	ClosesAt        time.Time
	StakedByOutcome map[OutcomeID]Amount
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
	stakedByOutcomes := make(map[OutcomeID]Amount, len(outcomes))

	for _, outcomes := range outcomes {
		stakedByOutcomes[outcomes], _ = NewAmount(0)
	}

	pool := &Pool{
		ID:              poolID,
		Creator:         creator,
		Resolver:        resolver,
		Question:        question,
		Outcomes:        slices.Clone(outcomes),
		State:           Open,
		ClosesAt:        closesAt,
		StakedByOutcome: stakedByOutcomes,
	}

	return pool, nil
}

func (p *Pool) PlaceBet(account AccountID, outcome OutcomeID, amount Amount) error {
	stakedAmount, ok := p.StakedByOutcome[outcome]
	if !ok {
		return fmt.Errorf("%w: %v, valid outcomes: %v", ErrUnknownOutcome, outcome, p.Outcomes)
	}
	newAmount, err := stakedAmount.Add(amount)
	if err != nil {
		return ErrInvalidAmount
	}
	p.StakedByOutcome[outcome] = newAmount
	return nil
}

func (p Pool) TotalBetOnOutcome(outcome OutcomeID) (Amount, error) {
	amount, ok := p.StakedByOutcome[outcome]
	if !ok {
		return Amount{}, fmt.Errorf("%w: %v", ErrUnknownOutcome, outcome)
	}
	return amount, nil
}
