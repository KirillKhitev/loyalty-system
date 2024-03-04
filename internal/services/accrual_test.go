package accrual

import (
	"github.com/go-resty/resty/v2"
	"loyalty-system/internal/config"
	"loyalty-system/internal/models/money"
	"reflect"
	"testing"
)

func TestAccrualService_GetDataOrderFromAPI(t *testing.T) {
	type fields struct {
		client *resty.Client
	}
	type args struct {
		orderNumber string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   APIServiceResult
	}{
		{
			name: "1 positive",
			fields: fields{
				client: resty.New(),
			},
			args: args{
				orderNumber: "3086248659",
			},
			want: APIServiceResult{
				Code: 200,
				Response: DataOrder{
					Order:   "3086248659",
					Status:  "PROCESSED",
					Accrual: money.Money(70000),
				},
			},
		},
		{
			name: "2 negative",
			fields: fields{
				client: resty.New(),
			},
			args: args{
				orderNumber: "53086248659",
			},
			want: APIServiceResult{
				Code: 204,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ac := &AccrualService{
				client: tt.fields.client,
			}

			config.Config.AccrualAddr = "http://localhost:8080"

			if got := ac.GetDataOrderFromAPI(tt.args.orderNumber); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetDataOrderFromAPI() = %v, want %v", got, tt.want)
			}
		})
	}
}
