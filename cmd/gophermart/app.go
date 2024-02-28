package main

import (
	"context"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-resty/resty/v2"
	"log"
	"loyalty-system/internal/config"
	"loyalty-system/internal/gzip"
	"loyalty-system/internal/handlers"
	"loyalty-system/internal/models/orders"
	accrual "loyalty-system/internal/services"
	"loyalty-system/internal/store"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
)

type app struct {
	store             store.Store
	server            http.Server
	ordersAccrualChan chan string
	pauseUpdatersChan chan int
	closeUpdatersChan chan struct{}
	wg                *sync.WaitGroup
	client            *resty.Client
}

func newApp(s store.Store) *app {
	instance := &app{
		store:             s,
		ordersAccrualChan: make(chan string, config.Config.AccrualUpdatersCount),
		pauseUpdatersChan: make(chan int, config.Config.AccrualUpdatersCount),
		closeUpdatersChan: make(chan struct{}),
		wg:                &sync.WaitGroup{},
		client:            resty.New(),
	}

	return instance
}

func (a *app) Close() error {
	if err := a.shutdownServer(); err != nil {
		return fmt.Errorf("error by Server shutdown: %w", err)
	}

	a.stopUpdaters()

	close(a.ordersAccrualChan)

	if err := a.store.Close(); err != nil {
		return fmt.Errorf("error by closing Store: %w", err)
	}

	log.Println("Store graceful shutdown complete.")

	return nil
}

func (a *app) stopUpdaters() {
	log.Println("Waiting closing all updaters")

	close(a.closeUpdatersChan)
	a.wg.Wait()

	log.Println("All updaters are stoped!")
}

func (a *app) StartServer() error {
	log.Printf("Running server: %v", config.Config)

	r := chi.NewRouter()

	r.Route("/api/user", func(r chi.Router) {
		r.Route("/balance", func(r chi.Router) {
			r.Handle("/", &handlers.Balance{
				Store: a.store,
			})
			r.Handle("/withdraw", &handlers.Withdraw{
				Store: a.store,
			})
		})
		r.Handle("/register", &handlers.Register{
			Store: a.store,
		})
		r.Handle("/login", &handlers.Login{
			Store: a.store,
		})
		r.Handle("/orders", &handlers.Orders{
			Store: a.store,
		})
		r.Handle("/withdrawals", &handlers.Withdrawals{
			Store: a.store,
		})
	})

	handler := gzip.Middleware(r)

	a.server = http.Server{
		Addr:    config.Config.AddrRun,
		Handler: handler,
	}

	return a.server.ListenAndServe()
}

func (a *app) shutdownServer() error {
	shutdownCtx, shutdownRelease := context.WithCancel(context.TODO())
	defer shutdownRelease()

	if err := a.server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("HTTP shutdown error: %w", err)
	}

	log.Println("HTTP graceful shutdown complete.")

	return nil
}

func (a *app) StartObserverStore(ctx context.Context) {
	for w := 1; w <= config.Config.AccrualUpdatersCount; w++ {
		a.wg.Add(1)
		go a.updater(ctx, w)
	}

	ticker := time.Tick(time.Second * time.Duration(config.Config.AccrualInterval))

	for {
		<-ticker

		ordersList, err := a.store.GetNewOrders(ctx)
		if err != nil {
			log.Printf("error by get new orders - %s", err)
			continue
		}

		for _, order := range ordersList {
			a.ordersAccrualChan <- order.Number
		}
	}
}

func (a *app) updater(ctx context.Context, idUpdater int) {
	for {
		select {
		case <-a.closeUpdatersChan:
			a.wg.Done()
			log.Printf("Stoped Updater #%d", idUpdater)
			return
		case delay := <-a.pauseUpdatersChan:
			log.Printf("Paused Updater #%d for %d seconds", idUpdater, delay)
			time.Sleep(time.Duration(delay) * time.Second)
			log.Printf("Restarted Updater #%d", idUpdater)
		case orderNumber := <-a.ordersAccrualChan:
			log.Printf("Updater #%d get order %s", idUpdater, orderNumber)

			if err := a.updateAccrualOrder(ctx, orderNumber, idUpdater); err != nil {
				log.Println(err)
				continue
			}
		}
	}
}

func (a *app) CatchTerminateSignal() error {
	terminateSignals := make(chan os.Signal, 1)

	signal.Notify(terminateSignals, syscall.SIGINT, syscall.SIGTERM)

	<-terminateSignals

	if err := a.Close(); err != nil {
		return err
	}

	log.Println("Terminate app complete")

	return nil
}

func (a *app) updateAccrualOrder(ctx context.Context, orderNumber string, idUpdater int) error {
	responseJSON := accrual.DataOrder{}

	url := fmt.Sprintf("http://%s%s", config.Config.AccrualAddr, orderNumber)

	if err := a.store.ChangeStatusOrder(ctx, orderNumber, orders.StatusList.Processing); err != nil {
		return fmt.Errorf("error by change status order '%s' - %s", orderNumber, err)
	}

	var responseErr accrual.ApiError

	response, err := a.client.R().
		SetError(&responseErr).
		SetResult(&responseJSON).
		Get(url)

	if err != nil {
		return fmt.Errorf("Updater %d, get data for Order %s, err: %v", idUpdater, orderNumber, responseErr)
	}

	switch response.StatusCode() {
	case http.StatusNoContent:
		log.Printf("Заказ %s не зарегистрирован в системе отчета!", orderNumber)
		if errChangeStatus := a.store.ChangeStatusOrder(ctx, orderNumber, orders.StatusList.Invalid); err != nil {
			return fmt.Errorf("error by change status order '%s' - %s", orderNumber, errChangeStatus)
		}
	case http.StatusInternalServerError:
		return fmt.Errorf("internal server error")
	case http.StatusOK:
		log.Printf("Успешно сходили в сервис расчета баллов по заказу '%s'", orderNumber)
		return a.updateOrderByAccrualDataOrder(ctx, responseJSON)
	case http.StatusTooManyRequests:
		return a.pauseUpdaters(response.Header().Get("Retry-After"))
	}

	return nil
}

func (a *app) pauseUpdaters(retryAfter string) error {
	log.Printf("Слишком много запросов! Нужно подождать %s секунд", retryAfter)

	value, err := strconv.Atoi(retryAfter)
	if err != nil {
		return fmt.Errorf("wrong value to Retry-After - %s", err)
	}

	for i := 1; i <= config.Config.AccrualUpdatersCount; i++ {
		a.pauseUpdatersChan <- value
	}

	return nil
}

func (a *app) updateOrderByAccrualDataOrder(ctx context.Context, data accrual.DataOrder) error {
	log.Printf("Ответ от сервиса: %v", data)

	switch data.Status {
	case accrual.OrderStatusList.Invalid:
		err := a.store.ChangeStatusOrder(ctx, data.Order, orders.StatusList.Invalid)
		if err != nil {
			return fmt.Errorf("error by change status order '%s' - %s", data.Order, err)
		}
	case accrual.OrderStatusList.Registered:
		fallthrough
	case accrual.OrderStatusList.Processing:
		err := a.store.ChangeStatusOrder(ctx, data.Order, orders.StatusList.New)
		if err != nil {
			return fmt.Errorf("error by change status order '%s' - %s", data.Order, err)
		}
	case accrual.OrderStatusList.Processed:
		err := a.store.UpdateStatusAndAccrualOrder(ctx, data)
		if err != nil {
			return fmt.Errorf("error by update accrual order '%s' - %s", data.Order, err)
		}
	}

	return nil
}
