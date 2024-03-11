package observer

import (
	"context"
	"fmt"
	"log"
	"loyalty-system/internal/config"
	"loyalty-system/internal/models/orders"
	accrual "loyalty-system/internal/services"
	"loyalty-system/internal/store"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type Observer struct {
	store             store.Store
	ordersAccrualChan chan string
	pauseUpdatersChan chan int
	closeUpdatersChan chan struct{}
	wg                *sync.WaitGroup
	accrualService    *accrual.AccrualService
	pauseUpdatersFlag bool
}

func NewObserver(s store.Store) *Observer {
	observer := &Observer{
		store:             s,
		ordersAccrualChan: make(chan string, config.Config.AccrualUpdatersCount),
		pauseUpdatersChan: make(chan int, config.Config.AccrualUpdatersCount),
		closeUpdatersChan: make(chan struct{}),
		wg:                &sync.WaitGroup{},
		accrualService:    accrual.NewAccrualService(),
		pauseUpdatersFlag: false,
	}

	return observer
}

func (o *Observer) Close() {
	o.stopUpdaters()

	close(o.ordersAccrualChan)
}

func (o *Observer) Start(ctx context.Context) {
	o.startUpdaters(ctx)
	o.startObserverNewOrders(ctx)
}

func (o *Observer) startUpdaters(ctx context.Context) {
	for w := 1; w <= config.Config.AccrualUpdatersCount; w++ {
		o.wg.Add(1)
		go o.updater(ctx, w)
	}
}

func (o *Observer) startObserverNewOrders(ctx context.Context) {
	ticker := time.NewTicker(time.Second * time.Duration(config.Config.AccrualInterval))
	defer ticker.Stop()

	for {
		<-ticker.C

		ordersList, err := o.store.GetNewOrders(ctx)
		if err != nil {
			log.Printf("error by get new orders - %s", err)
			continue
		}

		for _, order := range ordersList {
			o.ordersAccrualChan <- order.Number
		}
	}
}

func (o *Observer) updater(ctx context.Context, idUpdater int) {
	for {
		select {
		case <-o.closeUpdatersChan:
			o.wg.Done()
			log.Printf("Stoped Updater #%d", idUpdater)
			return
		case delay := <-o.pauseUpdatersChan:
			if o.pauseUpdatersFlag {
				continue
			}

			go func(o *Observer) {
				log.Printf("Paused Updater #%d for %d seconds", idUpdater, delay)

				o.pauseUpdatersFlag = true
				time.Sleep(time.Duration(delay) * time.Second)
				o.pauseUpdatersFlag = false

				log.Printf("Restarted Updater #%d", idUpdater)
			}(o)
		default:
			select {
			case orderNumber := <-o.ordersAccrualChan:
				log.Printf("Updater #%d get order %s", idUpdater, orderNumber)

				if o.pauseUpdatersFlag {
					log.Printf("Updater #%d в ожидании, пока завершится пауза", idUpdater)
					continue
				}

				if err := o.UpdateAccrualOrder(ctx, orderNumber, idUpdater); err != nil {
					log.Println(err)
					continue
				}
			default:
			}
		}
	}
}

func (o *Observer) UpdateAccrualOrder(ctx context.Context, orderNumber string, idUpdater int) error {
	if err := o.store.ChangeStatusOrder(ctx, orderNumber, orders.StatusList.Processing); err != nil {
		return fmt.Errorf("error by change status order '%s' - %w", orderNumber, err)
	}

	data := o.accrualService.GetDataOrderFromAPI(ctx, orderNumber)
	log.Printf("Updater #%d сходил в сервис за баллами по заказу '%s': %v", idUpdater, orderNumber, data)

	if data.Error != nil {
		if err := o.store.ChangeStatusOrder(ctx, orderNumber, orders.StatusList.New); err != nil {
			return fmt.Errorf("error by change status order '%s' - %w", orderNumber, err)
		}

		return fmt.Errorf("updater %d, get data for Order %s, err: %w", idUpdater, orderNumber, data.Error)
	}

	switch data.Code {
	case http.StatusNoContent:
		log.Printf("Заказ %s не зарегистрирован в системе отчета!", orderNumber)
		if errChangeStatus := o.store.ChangeStatusOrder(ctx, orderNumber, orders.StatusList.Invalid); errChangeStatus != nil {
			return fmt.Errorf("error by change status order '%s' - %w", orderNumber, errChangeStatus)
		}
	case http.StatusInternalServerError:
		return fmt.Errorf("internal server error")
	case http.StatusOK:
		return o.updateOrderByAccrualDataOrder(ctx, data.Response)
	case http.StatusTooManyRequests:
		return o.pauseUpdaters(data.RetryAfterHeader)
	}

	return nil
}

func (o *Observer) updateOrderByAccrualDataOrder(ctx context.Context, data orders.DataOrder) error {
	switch data.Status {
	case accrual.OrderStatusList.Invalid:
		err := o.store.ChangeStatusOrder(ctx, data.Order, orders.StatusList.Invalid)
		if err != nil {
			return fmt.Errorf("error by change status order '%s' - %w", data.Order, err)
		}
	case accrual.OrderStatusList.Registered, accrual.OrderStatusList.Processing:
		err := o.store.ChangeStatusOrder(ctx, data.Order, orders.StatusList.New)
		if err != nil {
			return fmt.Errorf("error by change status order '%s' - %w", data.Order, err)
		}
	case accrual.OrderStatusList.Processed:
		err := o.store.UpdateStatusAndAccrualOrder(ctx, data)
		if err != nil {
			return fmt.Errorf("error by update accrual order '%s' - %w", data.Order, err)
		}
	}

	return nil
}

func (o *Observer) stopUpdaters() {
	log.Println("Waiting closing all updaters")

	close(o.closeUpdatersChan)
	o.wg.Wait()

	log.Println("All updaters are stopped!")
}

func (o *Observer) pauseUpdaters(retryAfter string) error {
	log.Printf("Слишком много запросов! Нужно подождать %s секунд", retryAfter)

	value, err := strconv.Atoi(retryAfter)
	if err != nil {
		return fmt.Errorf("wrong value to Retry-After - %w", err)
	}

	for i := 1; i <= config.Config.AccrualUpdatersCount; i++ {
		o.pauseUpdatersChan <- value
	}

	return nil
}
