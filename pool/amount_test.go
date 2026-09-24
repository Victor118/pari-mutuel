package pool_test

import (
	"errors"
	"testing"

	"github.com/Victor118/pari-mutuel/pool"
)

func TestNewAmount_Negative_ReturnsError(t *testing.T) {
	_, err := pool.NewAmount(-50)
	if !errors.Is(err, pool.ErrInvalidAmount) {
		t.Errorf("err = %v, want ErrInvalidAmount", err)
	}
}
