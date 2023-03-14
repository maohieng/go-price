package price

// MutablePrice is used for any package that
// does not obey the role of marshal/unmarshal
// such as [cloud.google.com/go/firestore] package.
// Use with your own risk.
type MutablePrice struct {
	Price    `json:"-" firestore:"-" db:"-"`
	Amount   float64 `json:"amount" firestore:"amount" db:"amount"`
	Currency string  `json:"currency,omitempty" firestore:"currency" db:"currency"`
}

// NewMutablePrice creates a MutablePrice from a Price.
func NewMutablePrice(p Price) MutablePrice {
	return MutablePrice{
		Price:    p,
		Amount:   p.FloatAmount(),
		Currency: p.Currency(),
	}
}
