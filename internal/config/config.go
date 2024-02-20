package config

import (
	"flag"
	"os"
)

type Params struct {
	AddrRun            string
	DBConnectionString string
	AccrualAddr        string
}

var Config Params = Params{}

func (f *Params) Parse() {
	flag.StringVar(&f.AddrRun, "a", "localhost:8080", "address and port to run API")
	flag.StringVar(&f.DBConnectionString, "d", "", "string for connection to DB, format 'host=%s port=%s user=%s password=%s dbname=%s sslmode=%s'")
	flag.StringVar(&f.AccrualAddr, "r", "localhost:8080/api/orders/", "accrual system address")
	flag.Parse()

	if envRunAddr := os.Getenv(`RUN_ADDRESS`); envRunAddr != `` {
		f.AddrRun = envRunAddr
	}

	if envDBConnectionString := os.Getenv("DATABASE_URI"); envDBConnectionString != "" {
		f.DBConnectionString = envDBConnectionString
	}

	if envAccrualAddr := os.Getenv("ACCRUAL_SYSTEM_ADDRESS"); envAccrualAddr != "" {
		f.AccrualAddr = envAccrualAddr
	}
}
