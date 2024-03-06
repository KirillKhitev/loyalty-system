package observer

import (
	"context"
	"log"
	"loyalty-system/internal/config"
	accrual "loyalty-system/internal/services"
	"loyalty-system/internal/store"
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
	pauseUpdaters     bool
}

func NewObserver(s store.Store) *Observer {
	pauseUpdatersChan := make(chan int, config.Config.AccrualUpdatersCount)

	observer := &Observer{
		store:             s,
		ordersAccrualChan: make(chan string, config.Config.AccrualUpdatersCount),
		pauseUpdatersChan: pauseUpdatersChan,
		closeUpdatersChan: make(chan struct{}),
		wg:                &sync.WaitGroup{},
		accrualService:    accrual.NewAccrualService(s, pauseUpdatersChan),
		pauseUpdaters:     false,
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
	ticker := time.Tick(time.Second * time.Duration(config.Config.AccrualInterval))

	for {
		<-ticker

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
			if o.pauseUpdaters {
				continue
			}

			go func(o *Observer) {
				log.Printf("Paused Updater #%d for %d seconds", idUpdater, delay)

				o.pauseUpdaters = true
				time.Sleep(time.Duration(delay) * time.Second)
				o.pauseUpdaters = false

				log.Printf("Restarted Updater #%d", idUpdater)
			}(o)
		default:
			select {
			case orderNumber := <-o.ordersAccrualChan:
				log.Printf("Updater #%d get order %s", idUpdater, orderNumber)

				if o.pauseUpdaters {
					log.Printf("Updater #%d в ожидании, пока завершится пауза", idUpdater)
					continue
				}

				if err := o.accrualService.UpdateAccrualOrder(ctx, orderNumber, idUpdater); err != nil {
					log.Println(err)
					continue
				}
			default:
			}
		}
	}
}

func (o *Observer) stopUpdaters() {
	log.Println("Waiting closing all updaters")

	close(o.closeUpdatersChan)
	o.wg.Wait()

	log.Println("All updaters are stopped!")
}
