package main

import (
	"context"
	"fmt"
	"log"
	"loyalty-system/internal/store"
	"net/http"
)

type app struct {
	store  store.Store
	server http.Server
}

func newApp(s store.Store) *app {
	instance := &app{
		store: s,
	}

	return instance
}

func (a *app) Close() error {
	if err := a.shutdownServer(); err != nil {
		return err
	}

	if err := a.store.Close(); err != nil {
		return err
	}

	log.Println("Store graceful shutdown complete.")

	return nil
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
