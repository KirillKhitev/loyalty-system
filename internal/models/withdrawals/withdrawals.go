package withdrawals

import (
	"encoding/json"
	"github.com/beevik/guid"
	"loyalty-system/internal/models/money"
	"time"
)

type Withdraw struct {
	ID            string      `json:"-"`
	OrderID       string      `json:"-"`
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

func NewWithdraw(orderID, orderNumber string, sum money.Money) *Withdraw {
	withdraw := &Withdraw{
		ID:            guid.NewString(),
		OrderID:       orderID,
		Order:         orderNumber,
		Sum:           sum,
		ProcessedDate: time.Now(),
	}

	return withdraw
}
