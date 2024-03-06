package accrual

import (
	"context"
	"fmt"
	"github.com/go-resty/resty/v2"
	"log"
	"loyalty-system/internal/config"
	"loyalty-system/internal/models/orders"
	"loyalty-system/internal/store"
	"net/http"
	"strconv"
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

type APIError struct {
	Code      int       `json:"code"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

const APIUrl = "api/orders"
const APITimeout = 5

type AccrualService struct {
	client            *resty.Client
	store             store.Store
	pauseUpdatersChan chan int
}

func NewAccrualService(s store.Store, p chan int) *AccrualService {
	return &AccrualService{
		client:            resty.New(),
		store:             s,
		pauseUpdatersChan: p,
	}
}

type APIServiceResult struct {
	Code             int
	RetryAfterHeader string
	Response         orders.DataOrder
	Error            error
}

func (ac *AccrualService) GetDataOrderFromAPI(ctx context.Context, orderNumber string) APIServiceResult {
	result := APIServiceResult{}

	url := fmt.Sprintf("%s/%s/%s", config.Config.AccrualAddr, APIUrl, orderNumber)

	var responseErr APIError

	contextWithTimeout, cancel := context.WithTimeout(ctx, time.Duration(APITimeout*time.Second))
	defer cancel()

	response, err := ac.client.R().
		SetContext(contextWithTimeout).
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

func (ac *AccrualService) UpdateAccrualOrder(ctx context.Context, orderNumber string, idUpdater int) error {
	if err := ac.store.ChangeStatusOrder(ctx, orderNumber, orders.StatusList.Processing); err != nil {
		return fmt.Errorf("error by change status order '%s' - %w", orderNumber, err)
	}

	data := ac.GetDataOrderFromAPI(ctx, orderNumber)
	log.Printf("Updater #%d сходил в сервис за баллами по заказу '%s': %v", idUpdater, orderNumber, data)

	if data.Error != nil {
		if err := ac.store.ChangeStatusOrder(ctx, orderNumber, orders.StatusList.New); err != nil {
			return fmt.Errorf("error by change status order '%s' - %w", orderNumber, err)
		}

		return fmt.Errorf("updater %d, get data for Order %s, err: %w", idUpdater, orderNumber, data.Error)
	}

	switch data.Code {
	case http.StatusNoContent:
		log.Printf("Заказ %s не зарегистрирован в системе отчета!", orderNumber)
		if errChangeStatus := ac.store.ChangeStatusOrder(ctx, orderNumber, orders.StatusList.Invalid); errChangeStatus != nil {
			return fmt.Errorf("error by change status order '%s' - %w", orderNumber, errChangeStatus)
		}
	case http.StatusInternalServerError:
		return fmt.Errorf("internal server error")
	case http.StatusOK:
		return ac.updateOrderByAccrualDataOrder(ctx, data.Response)
	case http.StatusTooManyRequests:
		return ac.pauseUpdaters(data.RetryAfterHeader)
	}

	return nil
}

func (ac *AccrualService) updateOrderByAccrualDataOrder(ctx context.Context, data orders.DataOrder) error {
	switch data.Status {
	case OrderStatusList.Invalid:
		err := ac.store.ChangeStatusOrder(ctx, data.Order, orders.StatusList.Invalid)
		if err != nil {
			return fmt.Errorf("error by change status order '%s' - %w", data.Order, err)
		}
	case OrderStatusList.Registered, OrderStatusList.Processing:
		err := ac.store.ChangeStatusOrder(ctx, data.Order, orders.StatusList.New)
		if err != nil {
			return fmt.Errorf("error by change status order '%s' - %w", data.Order, err)
		}
	case OrderStatusList.Processed:
		err := ac.store.UpdateStatusAndAccrualOrder(ctx, data)
		if err != nil {
			return fmt.Errorf("error by update accrual order '%s' - %w", data.Order, err)
		}
	}

	return nil
}

func (ac *AccrualService) pauseUpdaters(retryAfter string) error {
	log.Printf("Слишком много запросов! Нужно подождать %s секунд", retryAfter)

	value, err := strconv.Atoi(retryAfter)
	if err != nil {
		return fmt.Errorf("wrong value to Retry-After - %w", err)
	}

	for i := 1; i <= config.Config.AccrualUpdatersCount; i++ {
		ac.pauseUpdatersChan <- value
	}

	return nil
}
