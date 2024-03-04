package orders

import (
	"github.com/stretchr/testify/require"
	"loyalty-system/internal/models/money"
	"testing"
	"time"
)

func TestNewOrder(t *testing.T) {
	type args struct {
		number string
		userID string
		status string
	}
	tests := []struct {
		name string
		args args
		want Order
	}{
		{
			name: "1 positive",
			args: args{
				number: "111222",
				userID: "1",
				status: "NEW",
			},
			want: Order{
				Number: "111222",
				UserID: "1",
				Status: "NEW",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := NewOrder(tt.args.number, tt.args.userID, tt.args.status)

			if order.Number != tt.want.Number || order.UserID != tt.want.UserID || order.Status != tt.want.Status {
				t.Errorf("wrong creating Order")
			}
		})
	}
}

func TestOrder_MarshalJSON(t *testing.T) {
	type fields struct {
		ID           string
		Number       string
		UserID       string
		Status       string
		Accrual      money.Money
		UploadedDate time.Time
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr bool
	}{
		{
			name: "1 positive",
			fields: fields{
				ID:           "1",
				Number:       "1111",
				UserID:       "2",
				Status:       "NEW",
				Accrual:      1000,
				UploadedDate: time.Date(2024, time.November, 10, 10, 10, 10, 0, time.UTC),
			},
			want: "{\"number\": \"1111\", \"status\": \"NEW\", \"accrual\":10, \"uploaded_at\": \"2024-11-10T10:10:10Z\"}",
		},
		{
			name: "2 positive",
			fields: fields{
				ID:           "2",
				Number:       "2222",
				UserID:       "3",
				Status:       "NEW",
				Accrual:      0,
				UploadedDate: time.Date(2024, time.November, 13, 10, 10, 10, 0, time.UTC),
			},
			want: "{\"number\": \"2222\", \"status\": \"NEW\", \"uploaded_at\": \"2024-11-13T10:10:10Z\"}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := Order{
				ID:           tt.fields.ID,
				Number:       tt.fields.Number,
				UserID:       tt.fields.UserID,
				Status:       tt.fields.Status,
				Accrual:      tt.fields.Accrual,
				UploadedDate: tt.fields.UploadedDate,
			}
			body, _ := o.MarshalJSON()

			require.JSONEq(t, tt.want, string(body))
		})
	}
}

func TestValidateNumber(t *testing.T) {
	type args struct {
		number uint64
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "1 negative",
			args: args{
				number: 123123123123,
			},
			want: false,
		},
		{
			name: "2 positive",
			args: args{
				number: 3086248659,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateNumber(tt.args.number); got != tt.want {
				t.Errorf("ValidateNumber() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_checkSum(t *testing.T) {
	type args struct {
		number uint64
	}
	tests := []struct {
		name string
		args args
		want uint64
	}{
		{
			name: "1 positive",
			args: args{
				number: 3086248659,
			},
			want: 9,
		},
		{
			name: "2 positive",
			args: args{
				number: 1335278652,
			},
			want: 8,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checkSum(tt.args.number); got != tt.want {
				t.Errorf("checkSum() = %v, want %v", got, tt.want)
			}
		})
	}
}
