package pool_test

import (
	"errors"
	"math"
	"testing"

	"github.com/Victor118/pari-mutuel/pool"
)

func mustPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Error("expected a panic, got none")
		}
	}()
	fn()
}

func TestNewAmount_Negative_ReturnsError(t *testing.T) {

	_, err := pool.NewAmount(-50)
	if !errors.Is(err, pool.ErrInvalidAmount) {
		t.Errorf("err = %v, want ErrInvalidAmount", err)
	}
}

func TestNewAmount_JustBelowZero(t *testing.T) {

	_, err := pool.NewAmount(-1)
	if !errors.Is(err, pool.ErrInvalidAmount) {
		t.Errorf("err = %v, want ErrInvalidAmount", err)
	}
}

func TestNewAmount_Zero_IsValid(t *testing.T) {

	got, err := pool.NewAmount(0)
	if err != nil {
		t.Fatalf("NewAmount(0) refused : %v", err)
	}
	zero := pool.Amount{}
	if got != zero {
		t.Errorf("NewAmount(0) = %v, want the zero value", got)
	}
}

func TestNewAmount_MinInt64_ReturnsError(t *testing.T) {

	if _, err := pool.NewAmount(math.MinInt64); !errors.Is(err, pool.ErrInvalidAmount) {
		t.Errorf("err = %v, want ErrInvalidAmount", err)
	}
}

func TestAmount_Add_AtMaxInt64_IsValid(t *testing.T) {

	max := mustAmount(t, math.MaxInt64)

	if got := max.Add(mustAmount(t, 0)); got != max {
		t.Errorf("Add(Zero()) = %v, want %v", got, max)
	}
}

func TestAmount_Add_BeyondMaxInt64_Panics(t *testing.T) {

	max := mustAmount(t, math.MaxInt64)

	mustPanic(t, func() { max.Add(mustAmount(t, 1)) })
}
