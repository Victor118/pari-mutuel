package pool

type Amount struct {
	cents int64
}

func Zero() Amount {
	return Amount{}
}

func NewAmount(cents int64) (Amount, error) {
	if cents < 0 {
		return Amount{}, ErrInvalidAmount
	}
	return Amount{
		cents: cents,
	}, nil
}

func (a Amount) Add(amount Amount) Amount {
	cents := a.cents + amount.cents
	if cents < 0 {

		panic("pool: dépassement de capacité sur Amount.Add")
	}
	return Amount{cents: cents}

}
