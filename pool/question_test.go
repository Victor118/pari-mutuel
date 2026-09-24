package pool_test

import (
	"errors"
	"testing"

	"github.com/Victor118/pari-mutuel/pool"
)

func TestNewQuestion_Empty_ReturnsError(t *testing.T) {

	_, err := pool.NewQuestion("")
	if !errors.Is(err, pool.ErrEmptyQuestion) {
		t.Errorf("err = %v, want ErrEmptyQuestion", err)
	}

}
func TestNewQuestion_Blank_ReturnsError(t *testing.T) {

	_, err := pool.NewQuestion(" ")
	if !errors.Is(err, pool.ErrEmptyQuestion) {
		t.Errorf("err = %v, want ErrEmptyQuestion", err)
	}

}
