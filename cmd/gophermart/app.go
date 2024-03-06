package main

import (
	"context"
	"fmt"
	"github.com/go-chi/chi/v5"
	"log"
	"loyalty-system/internal/config"
	"loyalty-system/internal/gzip"
	"loyalty-system/internal/handlers"
	"loyalty-system/internal/observer"
	"loyalty-system/internal/store"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

type app struct {
	store    store.Store
	server   http.Server
	observer *observer.Observer
}

func newApp(s store.Store) *app {
	instance := &app{
		store:    s,
		observer: observer.NewObserver(s),
	}

	return instance
}

func (a *app) Close() error {
	if err := a.shutdownServer(); err != nil {
		return fmt.Errorf("error by Server shutdown: %w", err)
	}

	a.observer.Close()

	if err := a.store.Close(); err != nil {
		return fmt.Errorf("error by closing Store: %w", err)
	}

	log.Println("Store graceful shutdown complete.")

	return nil
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
	a.observer.Start(ctx)
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
