package config

import (
	"flag"
	"log"
	"os"
	"strconv"
)

type Params struct {
	AddrRun              string
	DBConnectionString   string
	AccrualAddr          string
	AccrualInterval      int
	AccrualUpdatersCount int
}

var Config Params = Params{}

func (f *Params) Parse() {
	flag.StringVar(&f.AddrRun, "a", "localhost:45665", "address and port to run API")
	flag.StringVar(&f.DBConnectionString, "d", "", "string for connection to DB, format 'host=%s port=%s user=%s password=%s dbname=%s sslmode=%s'")
	flag.StringVar(&f.AccrualAddr, "r", "localhost:8080", "accrual system address")
	flag.IntVar(&f.AccrualInterval, "i", 1, "interval update accruals")
	flag.IntVar(&f.AccrualUpdatersCount, "u", 5, "count of updaters accruals")
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

	if envAccrualInterval := os.Getenv("ACCRUAL_INTERVAL"); envAccrualInterval != "" {
		if val, err := strconv.Atoi(envAccrualInterval); err == nil {
			f.AccrualInterval = val
		} else {
			log.Printf("wrong value environment ACCRUAL_INTERVAL: %s", envAccrualInterval)
		}
	}

	if envAccrualUpdatersCount := os.Getenv("ACCRUAL_UPDATERS_COUNT"); envAccrualUpdatersCount != "" {
		if val, err := strconv.Atoi(envAccrualUpdatersCount); err == nil {
			f.AccrualUpdatersCount = val
		} else {
			log.Printf("wrong value environment ACCRUAL_UPDATERS_COUNT: %s", envAccrualUpdatersCount)
		}
	}
}
