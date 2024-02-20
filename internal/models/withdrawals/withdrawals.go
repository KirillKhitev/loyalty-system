package withdrawals

import (
	"encoding/json"
	"github.com/beevik/guid"
	"loyalty-system/internal/models/money"
	"time"
)

type Withdraw struct {
	Id            string      `json:"-"`
	OrderId       string      `json:"-"`
	Order         string      `json:"order"`
	Sum           money.Money `json:"sum"`
	ProcessedDate time.Time   `json:"processed_at"`
}

func (w Withdraw) MarshalJSON() ([]byte, error) {
	type WithdrawAlias Withdraw

	aliasValue := struct {
		WithdrawAlias
		ProcessedDate string `json:"processed_at"`
	}{
		WithdrawAlias: WithdrawAlias(w),
		ProcessedDate: w.ProcessedDate.Format(time.RFC3339),
	}

	return json.Marshal(aliasValue)
}

func NewWithdraw(orderId, orderNumber string, sum money.Money) *Withdraw {
	withdraw := &Withdraw{
		Id:            guid.NewString(),
		OrderId:       orderId,
		Order:         orderNumber,
		Sum:           sum,
		ProcessedDate: time.Now(),
	}

	return withdraw
}
