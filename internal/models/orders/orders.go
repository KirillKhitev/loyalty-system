package orders

import (
	"encoding/json"
	"github.com/beevik/guid"
	"loyalty-system/internal/models/money"
	"time"
)

type Order struct {
	Id           string      `json:"-"`
	Number       string      `json:"number"`
	UserId       string      `json:"-"`
	Status       string      `json:"status"`
	Accrual      money.Money `json:"accrual,omitempty"`
	UploadedDate time.Time   `json:"uploaded_at"`
}

func ValidateNumber(number uint64) bool {
	return (number%10+checkSum(number/10))%10 == 0
}

func checkSum(number uint64) uint64 {
	var luhn uint64

	for i := 0; number > 0; i++ {
		cur := number % 10

		if i%2 == 0 {
			cur = cur * 2
			if cur > 9 {
				cur = cur%10 + cur/10
			}
		}

		luhn += cur
		number = number / 10
	}
	return luhn % 10
}

func (o Order) MarshalJSON() ([]byte, error) {
	type OrderAlias Order

	aliasValue := struct {
		OrderAlias
		UploadedDate string `json:"uploaded_at"`
	}{
		OrderAlias:   OrderAlias(o),
		UploadedDate: o.UploadedDate.Format(time.RFC3339),
	}

	return json.Marshal(aliasValue)
}

func NewOrder(number string, userId string, status string) *Order {
	order := &Order{
		Id:           guid.NewString(),
		Number:       number,
		UserId:       userId,
		Status:       status,
		Accrual:      0,
		UploadedDate: time.Now(),
	}

	return order
}

type Status struct {
	New        string
	Processing string
	Invalid    string
	Processed  string
}

var StatusList = Status{
	New:        "NEW",
	Processing: "PROCESSING",
	Invalid:    "INVALID",
	Processed:  "PROCESSED",
}
