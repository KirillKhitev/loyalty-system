package accrual

import (
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
