package pool

import "strings"

// Question est l'énoncé d'un pari. Value object : immuable, non vide,
// constructible uniquement via NewQuestion.
type Question struct {
	value string
}

func NewQuestion(question string) (Question, error) {
	if strings.TrimSpace(question) == "" {
		return Question{}, ErrEmptyQuestion
	}
	return Question{value: question}, nil
}
