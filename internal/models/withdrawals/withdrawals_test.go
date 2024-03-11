package withdrawals

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"loyalty-system/internal/models/money"
	"testing"
	"time"
)

func TestWithdraw_MarshalJSON(t *testing.T) {
	type fields struct {
		ID            string
		OrderID       string
		Order         string
		Sum           money.Money
		ProcessedDate time.Time
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "1 positive",
			fields: fields{
				ID:            "1111",
				OrderID:       "234234fdg3453g",
				Order:         "3086248659",
				Sum:           money.Money(50045),
				ProcessedDate: time.Date(2024, time.November, 13, 10, 10, 10, 0, time.UTC),
			},
			want: "{\"order\": \"3086248659\", \"sum\": 500.45, \"processed_at\": \"2024-11-13T10:10:10Z\"}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := Withdraw{
				ID:            tt.fields.ID,
				OrderID:       tt.fields.OrderID,
				Order:         tt.fields.Order,
				Sum:           tt.fields.Sum,
				ProcessedDate: tt.fields.ProcessedDate,
			}
			got, _ := w.MarshalJSON()
			require.JSONEq(t, tt.want, string(got), fmt.Sprintf("Withdraw.MarshalJSON() expect %s, got %s", tt.want, got))
		})
	}
}
