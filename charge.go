package price

import "errors"

const (
	// ChargeTypeGiftCard  used as a charge type for gift cards
	ChargeTypeGiftCard = "giftcard"
	// ChargeTypeMain used as default for a Charge
	ChargeTypeMain = "main"
)

type (
	// Charge is a Amount of a certain Type. Charge is used as value object

	Charge struct {
		// Price that is paid, can be in a certain currency
		Price Price
		// Value of the "Price" in another (base) currency
		Value Price
		// Type of the charge - can be ChargeTypeMain or something else. Used to differentiate between different charges of a single thing
		Type string
		// Reference contains further information to distinguish charges of the same type
		Reference string
	}

	// Charges - Represents the Charges the product need to be paid with

	Charges struct {
		chargesByQualifier map[ChargeQualifier]Charge
	}

	// ChargeQualifier distinguishes charges by type and reference

	ChargeQualifier struct {
		// Type represents charge type
		Type string
		// Reference contains further information to distinguish charges of the same type
		Reference string
	}
)

// Add the given Charge to the current Charge and returns a new Charge
func (p Charge) Add(add Charge) (Charge, error) {
	if p.Type != add.Type {
		return Charge{}, errors.New("charge type mismatch")
	}
	newPrice, err := p.Price.Add(add.Price)
	if err != nil {
		return Charge{}, err
	}
	p.Price = newPrice

	newPrice, err = p.Value.Add(add.Value)
	if err != nil {
		return Charge{}, err
	}
	p.Value = newPrice
	return p, nil
}

// GetPayable rounds the charge
func (p Charge) GetPayable() Charge {
	p.Value = p.Value.GetPayable()
	p.Price = p.Price.GetPayable()
	return p
}

// Mul the given Charge and returns a new Charge
func (p Charge) Mul(qty int) Charge {
	p.Price = p.Price.Multiply(qty)
	p.Value = p.Value.Multiply(qty)
	return p
}

// NewCharges creates a new Charges object
func NewCharges(chargesByType map[string]Charge) *Charges {
	charges := addChargeQualifier(chargesByType)
	return &charges
}

// HasType returns a true if any charges include a charge with given type
func (c Charges) HasType(ctype string) bool {
	for qualifier := range c.chargesByQualifier {
		if qualifier.Type == ctype {
			return true
		}
	}
	return false
}

// GetByType returns a charge of given type. If it was not found a Zero amount
// is returned and the second return value is false
// sums up charges by a certain type if there are multiple
func (c Charges) GetByType(ctype string) (Charge, bool) {
	// guard in case type is not available
	if !c.HasType(ctype) {
		return Charge{}, false
	}
	result := Charge{
		Type: ctype,
	}
	// sum up all charges with certain type to one charge
	for qualifier, charge := range c.chargesByQualifier {
		if qualifier.Type == ctype {
			result, _ = result.Add(charge)
		}
	}
	return result, true
}

// HasChargeQualifier returns a true if any charges include a charge with given type
// and concrete key values provided by additional
func (c Charges) HasChargeQualifier(qualifier ChargeQualifier) bool {
	if _, ok := c.chargesByQualifier[qualifier]; ok {
		return true
	}
	return false
}

// GetByChargeQualifier returns a charge of given qualifier.
// If it was not found a Zero amount is returned and the second return value is false
func (c Charges) GetByChargeQualifier(qualifier ChargeQualifier) (Charge, bool) {
	// guard in case type is not available
	if !c.HasChargeQualifier(qualifier) {
		return Charge{}, false
	}

	if charge, ok := c.chargesByQualifier[qualifier]; ok {
		return charge, true
	}
	return Charge{}, false
}

// GetByChargeQualifierForced returns a charge of given qualifier.
// If it was not found a Zero amount is returned. This method might be useful to call in View (template) directly.
func (c Charges) GetByChargeQualifierForced(qualifier ChargeQualifier) Charge {
	result, ok := c.GetByChargeQualifier(qualifier)
	if !ok {
		return Charge{}
	}
	return result
}

// GetByTypeForced returns a charge of given type. If it was not found a Zero amount is returned.
// This method might be useful to call in View (template) directly where you need one return value
// sums up charges by a certain type if there are multiple
func (c Charges) GetByTypeForced(ctype string) Charge {
	result, ok := c.GetByType(ctype)
	if !ok {
		return Charge{}
	}
	return result
}

// GetAllCharges returns all charges
func (c Charges) GetAllCharges() map[ChargeQualifier]Charge {
	return c.chargesByQualifier
}

// GetAllByType returns all charges of type
func (c Charges) GetAllByType(ctype string) map[ChargeQualifier]Charge {
	chargesByType := make(map[ChargeQualifier]Charge)

	for qualifier, charge := range c.chargesByQualifier {
		if qualifier.Type == ctype {
			chargesByType[ChargeQualifier{
				qualifier.Type,
				qualifier.Reference,
			}] = charge
		}
	}

	return chargesByType
}

// Add returns new Charges with the given added
func (c Charges) Add(toadd Charges) Charges {
	if c.chargesByQualifier == nil {
		c.chargesByQualifier = make(map[ChargeQualifier]Charge)
	}
	for addk, addCharge := range toadd.chargesByQualifier {
		if existingCharge, ok := c.chargesByQualifier[addk]; ok {
			chargeSum, _ := existingCharge.Add(addCharge)
			c.chargesByQualifier[addk] = chargeSum.GetPayable()
		} else {
			c.chargesByQualifier[addk] = addCharge
		}
	}
	return c
}

// AddCharge returns new Charges with the given Charge added
func (c Charges) AddCharge(toadd Charge) Charges {
	if c.chargesByQualifier == nil {
		c.chargesByQualifier = make(map[ChargeQualifier]Charge)
	}
	qualifier := ChargeQualifier{
		Type:      toadd.Type,
		Reference: toadd.Reference,
	}
	if existingCharge, ok := c.chargesByQualifier[qualifier]; ok {
		chargeSum, _ := existingCharge.Add(toadd)
		c.chargesByQualifier[qualifier] = chargeSum.GetPayable()
	} else {
		c.chargesByQualifier[qualifier] = toadd
	}

	return c
}

// Mul returns new Charges with the given multiplied
func (c Charges) Mul(qty int) Charges {
	if c.chargesByQualifier == nil {
		return c
	}
	for t, charge := range c.chargesByQualifier {
		c.chargesByQualifier[t] = charge.Mul(qty)
	}
	return c
}

// Items returns all charges items
func (c Charges) Items() []Charge {
	var charges []Charge

	for _, charge := range c.chargesByQualifier {
		charges = append(charges, charge)
	}

	return charges
}

// addChargeQualifier parse string keys to charge qualifier for backwards compatibility
func addChargeQualifier(chargesByType map[string]Charge) Charges {
	withQualifier := make(map[ChargeQualifier]Charge)
	for chargeType, charge := range chargesByType {
		qualifier := ChargeQualifier{
			Type:      chargeType,
			Reference: charge.Reference,
		}
		withQualifier[qualifier] = charge
	}
	return Charges{chargesByQualifier: withQualifier}
}
