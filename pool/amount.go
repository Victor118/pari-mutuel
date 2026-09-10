package pool

import "fmt"

type Amount struct {
	cents int64
}

func NewAmount(cents int64) (Amount, error) {
	if cents < 0 {
		return Amount{}, ErrInvalidAmount
	}
	return Amount{
		cents: cents,
	}, nil
}

func (a Amount) Add(amount Amount) (Amount, error) {
	return NewAmount(a.cents + amount.cents)

}

// String rend le montant en unités principales : 12000 -> "120.00".
func (a Amount) String() string {
	return fmt.Sprintf("%d.%02d", a.cents/100, a.cents%100)
}
