package pool

type Currency string

type Amount struct {
	cents int64
}

type Money struct {
	amount   Amount
	currency Currency
}

func Zero(currency Currency) Money {
	return Money{
		amount:   Amount{},
		currency: currency,
	}
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

func NewMoney(amount Amount, currency Currency) Money {
	return Money{
		amount:   amount,
		currency: currency,
	}
}

func (m Money) Cents() int64 {
	return m.amount.cents
}
