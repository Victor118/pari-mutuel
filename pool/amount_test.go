package pool_test

import (
	"errors"
	"testing"

	"github.com/Victor118/pari-mutuel/pool"
)

func TestNewAmount_ErrorWhenNegative(t *testing.T) {
	_, err := pool.NewAmount(-50)
	if !errors.Is(err, pool.ErrInvalidAmount) {
		t.Errorf("err should be ErrNegativAmount but is : %v", err)
	}
}
