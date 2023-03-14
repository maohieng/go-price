package price

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"math"
	"math/big"
	"strings"
)

type (
	// Price is a Type that represents a Amount - it is immutable
	// DevHint: We use Amount and Charge as Value - so we do not pass pointers. (According to Go Wiki's code review comments page suggests passing by value when structs are small and likely to stay that way)
	Price struct {
		amount   big.Float `swaggertype:"string"`
		currency string
	}

	priceJSON struct {
		Amount   string `db:"amount,omitempty" firestore:"amount,omitempty" json:"amount,omitempty"`
		Currency string `db:"currency,omitempty" firestore:"currency,omitempty" json:"currency,omitempty"`
	}
)

const (
	// RoundingModeFloor use if you want to cut (round down)
	RoundingModeFloor = "floor"
	// RoundingModeCeil use if you want to round up always
	RoundingModeCeil = "ceil"
	// RoundingModeHalfUp round up if the discarded fraction is ≥ 0.5, otherwise round down. Default for GetPayable()
	RoundingModeHalfUp = "halfup"
	// RoundingModeHalfDown round up if the discarded fraction is > 0.5, otherwise round down.
	RoundingModeHalfDown = "halfdown"
)

// NewFromFloat - factory method
func NewFromFloat(amount float64, currency string) Price {
	return Price{
		amount:   *big.NewFloat(amount),
		currency: currency,
	}
}

// NewFromBigFloat - factory method
func NewFromBigFloat(amount big.Float, currency string) Price {
	return Price{
		amount:   amount,
		currency: currency,
	}
}

// NewZero Zero price
func NewZero(currency string) Price {
	return Price{
		amount:   *new(big.Float).SetInt64(0),
		currency: currency,
	}
}

// NewFromInt use to set money by smallest payable unit - e.g. to set 2.45 EUR you should use NewFromInt(245, 100, "EUR")
func NewFromInt(amount int64, precision int, currency string) Price {
	amountF := new(big.Float).SetInt64(amount)
	if precision == 0 {
		return Price{
			amount:   *new(big.Float).SetInt64(0),
			currency: currency,
		}
	}
	precicionF := new(big.Float).SetInt64(int64(precision))
	return Price{
		amount:   *new(big.Float).Quo(amountF, precicionF),
		currency: currency,
	}
}

func (p Price) String() string {
	bytes, _ := p.MarshalText()
	return string(bytes)
}

func (p Price) MarshalText() (text []byte, err error) {
	pj := &priceJSON{
		Amount:   p.amount.String(),
		Currency: p.currency,
	}
	return json.Marshal(pj)
}

func (p *Price) UnmarshalText(b []byte) error {
	pj := &priceJSON{}
	err := json.Unmarshal(b, pj)
	if err != nil {
		return err
	}

	am, _, err := new(big.Float).Parse(pj.Amount, 10)
	if err != nil {
		return err
	}

	p.amount = *am
	p.currency = pj.Currency

	return nil
}

// MarshalJSON implements interface required by json marshal
func (p Price) MarshalJSON() (data []byte, err error) {
	return p.MarshalText()
}

// UnmarshalJSON implements encode Unmarshaler
func (p *Price) UnmarshalJSON(data []byte) error {
	return p.UnmarshalText(data)
}

// MarshalBinary implements interface required by gob
func (p Price) MarshalBinary() (data []byte, err error) {
	return p.MarshalText()
}

// UnmarshalBinary implements interface required by gob.
// Modifies the receiver so it must take a pointer receiver!
func (p *Price) UnmarshalBinary(data []byte) error {
	return p.UnmarshalText(data)
}

// Value makes the Price struct implement the driver.Valuer interface. This method
// simply returns the JSON-encoded representation of the struct.
func (p Price) Value() (driver.Value, error) {
	return json.Marshal(p)
}

// Scan makes the Price struct implement the sql.Scanner interface. This method
// simply decodes a JSON-encoded value into the struct fields.
func (p *Price) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &p)
}

// Add the given price to the current price and returns a new price
func (p Price) Add(add Price) (Price, error) {
	newPrice, err := p.currencyGuard(add)
	if err != nil {
		return newPrice, err
	}
	newPrice.amount.Add(&p.amount, &add.amount)
	return newPrice, nil
}

// ForceAdd tries to add the given price to the current price - will not return errors
func (p Price) ForceAdd(add Price) Price {
	newPrice, err := p.currencyGuard(add)
	if err != nil {
		return p
	}
	newPrice.amount.Add(&p.amount, &add.amount)
	return newPrice
}

// currencyGuard is a common Guard that protects price calculations of prices with different currency.
// Robust: if original is Zero and the currencies are different we take the given currency
func (p Price) currencyGuard(check Price) (Price, error) {
	if p.currency == check.currency {
		return Price{
			currency: check.currency,
		}, nil
	}
	if p.IsZero() {
		return Price{
			currency: check.currency,
		}, nil
	}

	if check.IsZero() {
		return Price{
			currency: p.currency,
		}, nil
	}
	return NewZero(p.currency), errors.New("cannot calculate prices in different currencies")
}

// Discounted returns new price reduced by given percent
func (p Price) Discounted(percent float64) Price {
	newPrice := Price{
		currency: p.currency,
		amount:   *new(big.Float).Mul(&p.amount, big.NewFloat((100-percent)/100)),
	}
	return newPrice
}

// Taxed returns new price added with Tax (assuming current price is net)
func (p Price) Taxed(percent big.Float) Price {
	newPrice := Price{
		currency: p.currency,
		amount:   *new(big.Float).Add(&p.amount, p.TaxFromNet(percent).Amount()),
	}
	return newPrice
}

// TaxFromNet returns new price representing the tax amount (assuming the current price is net 100%)
func (p Price) TaxFromNet(percent big.Float) Price {
	quo := new(big.Float).Mul(&percent, &p.amount)
	newPrice := Price{
		currency: p.currency,
		amount:   *new(big.Float).Quo(quo, new(big.Float).SetInt64(100)),
	}
	return newPrice
}

// TaxFromGross returns new price representing the tax amount (assuming the current price is gross 100+percent)
func (p Price) TaxFromGross(percent big.Float) Price {
	quo := new(big.Float).Mul(&percent, &p.amount)
	percent100 := new(big.Float).Add(&percent, new(big.Float).SetInt64(100))
	newPrice := Price{
		currency: p.currency,
		amount:   *new(big.Float).Quo(quo, percent100),
	}
	return newPrice
}

// Sub the given price from the current price and returns a new price
// Sub using [big.Float.Sub]
func (p Price) Sub(sub Price) (Price, error) {
	newPrice, err := p.currencyGuard(sub)
	if err != nil {
		return newPrice, err
	}
	newPrice.amount.Sub(&p.amount, &sub.amount)
	return newPrice, nil
}

// Inverse returns the price multiplied with -1
func (p Price) Inverse() Price {
	p.amount = *new(big.Float).Mul(&p.amount, big.NewFloat(-1))
	return p
}

// Multiply returns a new price with the amount Multiply
func (p Price) Multiply(qty int) Price {
	newPrice := Price{
		currency: p.currency,
	}
	newPrice.amount.Mul(&p.amount, new(big.Float).SetInt64(int64(qty)))
	return newPrice
}

// Divided returns a new price with the amount Divided
func (p Price) Divided(qty int) Price {
	newPrice := Price{
		currency: p.currency,
	}
	if qty == 0 {
		return NewZero(p.currency)
	}
	newPrice.amount.Quo(&p.amount, new(big.Float).SetInt64(int64(qty)))
	return newPrice
}

// Equal compares the prices exact
func (p Price) Equal(cmp Price) bool {
	if p.currency != cmp.currency {
		return false
	}
	return p.amount.Cmp(&cmp.amount) == 0
}

// LikelyEqual compares the prices with some tolerance
func (p Price) LikelyEqual(cmp Price) bool {
	if p.currency != cmp.currency {
		return false
	}
	diff := new(big.Float).Sub(&p.amount, &cmp.amount)
	absDiff := new(big.Float).Abs(diff)
	return absDiff.Cmp(big.NewFloat(0.000000001)) == -1
}

// IsLessThen compares the current price with a given one
func (p Price) IsLessThen(cmp Price) bool {
	if p.currency != cmp.currency {
		return false
	}
	return p.amount.Cmp(&cmp.amount) == -1
}

// IsGreaterThen compares the current price with a given one
func (p Price) IsGreaterThen(cmp Price) bool {
	if p.currency != cmp.currency {
		return false
	}
	return p.amount.Cmp(&cmp.amount) == 1
}

// IsLessThenValue compares the price with a given amount value (assuming same currency)
func (p Price) IsLessThenValue(amount big.Float) bool {
	return p.amount.Cmp(&amount) == -1
}

// IsGreaterThenValue compares the price with a given amount value (assuming same currency)
func (p Price) IsGreaterThenValue(amount big.Float) bool {
	return p.amount.Cmp(&amount) == 1
}

// IsNegative returns true if the price represents a negative value
func (p Price) IsNegative() bool {
	return p.IsLessThenValue(*big.NewFloat(0.0))
}

// IsPositive returns true if the price represents a positive value
func (p Price) IsPositive() bool {
	return p.IsGreaterThenValue(*big.NewFloat(0.0))
}

// IsPayable returns true if the price represents a payable (rounded) value
func (p Price) IsPayable() bool {
	return p.GetPayable().Equal(p)
}

// IsZero returns true if the price represents zero value
func (p Price) IsZero() bool {
	return p.LikelyEqual(NewZero(p.Currency())) || p.LikelyEqual(NewFromFloat(0, p.Currency()))
}

// FloatAmount gets the current amount as float
func (p Price) FloatAmount() float64 {
	a, _ := p.amount.Float64()
	return a
}

// GetPayable rounds the price with the precision required by the currency in a price that can actually be paid
// e.g. an internal amount of 1,23344 will get rounded to 1,23
func (p Price) GetPayable() Price {
	mode, precision := p.payableRoundingPrecision()
	return p.GetPayableByRoundingMode(mode, precision)
}

// GetPayableByRoundingMode returns the price rounded you can pass the used rounding mode and precision
// Example for precision 100:
//
//	1.115 >  1.12 (RoundingModeHalfUp)  / 1.11 (RoundingModeFloor)
//	-1.115 > -1.11 (RoundingModeHalfUp) / -1.12 (RoundingModeFloor)
func (p Price) GetPayableByRoundingMode(mode string, precision int) Price {
	newPrice := Price{
		currency: p.currency,
	}

	amountForRound := new(big.Float).Copy(&p.amount)
	negative := int64(1)
	if p.IsNegative() {
		negative = -1
	}

	amountTruncatedFloat, _ := new(big.Float).Mul(amountForRound, p.precisionF(precision)).Float64()
	integerPart, fractionalPart := math.Modf(amountTruncatedFloat)
	amountTruncatedInt := int64(integerPart)
	valueAfterPrecision := (math.Round(fractionalPart*1000) / 100) * float64(negative)
	if amountTruncatedFloat >= float64(math.MaxInt64) {
		// will not work if we are already above MaxInt - so we return unrounded price:
		newPrice.amount = p.amount
		return newPrice
	}

	switch mode {
	case RoundingModeCeil:
		if negative == 1 && valueAfterPrecision > 0 {
			amountTruncatedInt = amountTruncatedInt + negative
		}
	case RoundingModeHalfUp:
		if valueAfterPrecision >= 5 {
			amountTruncatedInt = amountTruncatedInt + negative
		}
	case RoundingModeHalfDown:
		if valueAfterPrecision > 5 {
			amountTruncatedInt = amountTruncatedInt + negative
		}
	case RoundingModeFloor:
		if negative == -1 && valueAfterPrecision > 0 {
			amountTruncatedInt = amountTruncatedInt + negative
		}
	default:
		// nothing to round
	}

	amountRounded := new(big.Float).Quo(new(big.Float).SetInt64(amountTruncatedInt), p.precisionF(precision))
	newPrice.amount = *amountRounded
	return newPrice
}

// precisionF returns big.Float from int
func (p Price) precisionF(precision int) *big.Float {
	return new(big.Float).SetInt64(int64(precision))
}

// precisionF - 10 * n - n is the amount of decimal numbers after comma
// - can be currency specific (for now defaults to 2)
// - TODO - use currency configuration or registry
func (p Price) payableRoundingPrecision() (string, int) {
	if strings.ToLower(p.currency) == "miles" || strings.ToLower(p.currency) == "points" {
		return RoundingModeFloor, int(1)
	}
	return RoundingModeHalfUp, int(100)
}

// SplitInPayables returns "count" payable prices (each rounded) that in sum matches the given price
//   - Given a price of 12.456 (Payable 12,46)  - Splitted in 6 will mean: 6 * 2.076
//   - but having them payable requires rounding them each (e.g. 2.07) which would mean we have 0.03 difference (=12,45-6*2.07)
//   - so that the sum is as close as possible to the original value   in this case the correct return will be:
//   - 2.07 + 2.07+2.08 +2.08 +2.08 +2.08
func (p Price) SplitInPayables(count int) ([]Price, error) {
	if count <= 0 {
		return nil, errors.New("split must be higher than zero")
	}
	// guard clause invert negative values
	_, precision := p.payableRoundingPrecision()
	amount := p.GetPayable().Amount()
	// we have to invert negative numbers, otherwise split is not correct
	if p.IsNegative() {
		amount = p.GetPayable().Inverse().Amount()
	}
	amountToMatchFloat, _ := new(big.Float).Mul(amount, p.precisionF(precision)).Float64()
	amountToMatchInt := int64(amountToMatchFloat)

	splittedAmountModulo := amountToMatchInt % int64(count)
	splittedAmount := amountToMatchInt / int64(count)

	splittedAmounts := make([]int64, count)
	for i := 0; i < count; i++ {
		splittedAmounts[i] = splittedAmount
	}

	for i := 0; i < int(splittedAmountModulo); i++ {
		splittedAmounts[i] = splittedAmounts[i] + 1
	}

	prices := make([]Price, count)
	for i := 0; i < count; i++ {
		_, precision := p.payableRoundingPrecision()
		splittedAmount := splittedAmounts[i]
		// invert prices again to keep negative values
		if p.IsNegative() {
			splittedAmount *= -1
		}
		prices[i] = NewFromInt(splittedAmount, precision, p.Currency())
	}

	return prices, nil
}

// Clone returns a copy of the price - the amount gets Excat acc
func (p Price) Clone() Price {
	return Price{
		amount:   *new(big.Float).Set(&p.amount),
		currency: p.currency,
	}
}

// Currency returns currency
func (p Price) Currency() string {
	return p.currency
}

// Amount returns exact amount as bigFloat
func (p Price) Amount() *big.Float {
	return &p.amount
}

// SumAll returns new price with sum of all given prices
func SumAll(prices ...Price) (Price, error) {
	if len(prices) == 0 {
		return NewZero(""), errors.New("no price given")
	}
	result := prices[0].Clone()
	var err error
	for _, price := range prices[1:] {
		result, err = result.Add(price)
		if err != nil {
			return result, err
		}
	}
	return result, nil
}
