package main

import (
	"database/sql"
	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"log"
	"loyalty-system/internal/config"
	"loyalty-system/internal/gzip"
	"loyalty-system/internal/handlers"
	"loyalty-system/internal/store/pg"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	config.Config.Parse()

	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	conn, err := sql.Open("pgx", config.Config.DBConnectionString)
	if err != nil {
		return err
	}

	appInstance := newApp(pg.NewStore(conn))

	go startServer(appInstance)

	return catchTerminateSignal(appInstance)
}

func startServer(appInstance *app) error {
	log.Println("Running server", config.Config)

	r := chi.NewRouter()

	r.Route("/api/user", func(r chi.Router) {
		r.Route("/balance", func(r chi.Router) {
			r.Handle("/", &handlers.Balance{
				Store: appInstance.store,
			})
			r.Handle("/withdraw", &handlers.Withdraw{
				Store: appInstance.store,
			})
		})
		r.Handle("/register", &handlers.Register{
			Store: appInstance.store,
		})
		r.Handle("/login", &handlers.Login{
			Store: appInstance.store,
		})
		r.Handle("/orders", &handlers.Orders{
			Store: appInstance.store,
		})
		r.Handle("/withdrawals", &handlers.Withdrawals{
			Store: appInstance.store,
		})
	})

	handler := gzip.Middleware(r)

	appInstance.server = http.Server{
		Addr:    config.Config.AddrRun,
		Handler: handler,
	}

	return appInstance.server.ListenAndServe()
}

func catchTerminateSignal(appInstance *app) error {
	terminateSignals := make(chan os.Signal, 1)

	signal.Notify(terminateSignals, syscall.SIGINT, syscall.SIGTERM)

	<-terminateSignals

	appInstance.Close()

	log.Println("Terminate app complete")

	return nil
}
