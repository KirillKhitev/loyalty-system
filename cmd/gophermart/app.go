package main

import (
	"context"
	"fmt"
	"github.com/go-chi/chi/v5"
	"log"
	"loyalty-system/internal/config"
	"loyalty-system/internal/gzip"
	"loyalty-system/internal/handlers"
	accrual "loyalty-system/internal/services"
	"loyalty-system/internal/store"
	"net/http"
	"os"
	"os/signal"
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
	accrualService    *accrual.AccrualService
	pauseUpdaters     bool
}

func newApp(s store.Store) *app {
	pauseUpdatersChan := make(chan int, config.Config.AccrualUpdatersCount)

	instance := &app{
		store:             s,
		ordersAccrualChan: make(chan string, config.Config.AccrualUpdatersCount),
		pauseUpdatersChan: pauseUpdatersChan,
		closeUpdatersChan: make(chan struct{}),
		wg:                &sync.WaitGroup{},
		accrualService:    accrual.NewAccrualService(s, pauseUpdatersChan),
		pauseUpdaters:     false,
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
			go func(a *app) {
				log.Printf("Paused Updater #%d for %d seconds", idUpdater, delay)

				a.pauseUpdaters = true
				time.Sleep(time.Duration(delay) * time.Second)
				a.pauseUpdaters = false

				log.Printf("Restarted Updater #%d", idUpdater)
			}(a)
		default:
			select {
			case orderNumber := <-a.ordersAccrualChan:
				log.Printf("Updater #%d get order %s", idUpdater, orderNumber)

				if a.pauseUpdaters {
					log.Printf("Updater #%d в ожидании, пока завершится пауза", idUpdater)
					continue
				}

				if err := a.accrualService.UpdateAccrualOrder(ctx, orderNumber, idUpdater); err != nil {
					log.Println(err)
					continue
				}
			default:
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
