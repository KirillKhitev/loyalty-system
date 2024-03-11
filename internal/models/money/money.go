package money

import (
	"encoding/json"
)

type Money uint64

func (v Money) MarshalJSON() ([]byte, error) {
	return json.Marshal(float64(v) / 100)
}

func (v *Money) UnmarshalJSON(data []byte) error {
	var tmp float64

	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	*v = Money(tmp * 100)

	return nil
}
