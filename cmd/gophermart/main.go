package main

import (
	"context"
	"database/sql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"loyalty-system/internal/config"
	"loyalty-system/internal/store/pg"
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

	ctx := context.Background()
	appInstance := newApp(pg.NewStore(ctx, conn))

	go appInstance.StartServer()
	go appInstance.StartObserverStore(ctx)

	return appInstance.CatchTerminateSignal()
}
