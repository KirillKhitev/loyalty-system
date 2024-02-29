package accrual

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	"loyalty-system/internal/config"
	"loyalty-system/internal/models/money"
	"time"
)

type OrderStatus struct {
	Registered string
	Invalid    string
	Processing string
	Processed  string
}

var OrderStatusList = OrderStatus{
	Registered: "REGISTERED",
	Invalid:    "INVALID",
	Processing: "PROCESSING",
	Processed:  "PROCESSED",
}

type DataOrder struct {
	Order   string      `json:"order"`
	Status  string      `json:"status"`
	Accrual money.Money `json:"accrual,omitempty"`
}

type APIError struct {
	Code      int       `json:"code"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

const APIUrl = "api/orders"

type AccrualService struct {
	client *resty.Client
}

func NewAccrualService() *AccrualService {
	return &AccrualService{
		client: resty.New(),
	}
}

type APIServiceResult struct {
	Code             int
	RetryAfterHeader string
	Response         DataOrder
	Error            error
}

func (ac *AccrualService) GetDataOrderFromAPI(orderNumber string) APIServiceResult {
	result := APIServiceResult{}

	url := fmt.Sprintf("%s/%s/%s", config.Config.AccrualAddr, APIUrl, orderNumber)

	var responseErr APIError

	response, err := ac.client.R().
		SetError(&result.Error).
		SetResult(&result.Response).
		Get(url)

	if err != nil {
		result.Error = fmt.Errorf("%v", responseErr)
		return result
	}

	result.Code = response.StatusCode()
	result.RetryAfterHeader = response.Header().Get("Retry-After")

	return result
}
