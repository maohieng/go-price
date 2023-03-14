package price

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// Discount represents the amount of discount in Price or percentage.
// Percentage is priority used over Price.
// To get discounted Price, use Discounted func for percentage
//
//	or Sub func for Price.
type Discount struct {
	Price      Price `db:"price,omitempty" firestore:"price,omitempty" json:"price,omitempty"`
	Percentage int   `db:"percentage,omitempty" firestore:"percentage,omitempty" json:"percentage,omitempty"`
}

// Value makes the Discount struct implement the driver.Valuer interface. This method
// simply returns the JSON-encoded representation of the struct.
func (a Discount) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// Scan makes the Discount struct implement the sql.Scanner interface. This method
// simply decodes a JSON-encoded value into the struct fields.
func (a *Discount) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &a)
}
