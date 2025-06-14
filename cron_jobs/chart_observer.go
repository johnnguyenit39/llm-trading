package cronjobs

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/rs/zerolog/log"
)

type ChartObserver struct {
	resultChan chan int
	stopChan   chan struct{}
}

func NewChartObserver() *ChartObserver {
	return &ChartObserver{
		resultChan: make(chan int),
		stopChan:   make(chan struct{}),
	}
}

func (o *ChartObserver) StartChartObserver() {
	go o.run()
}

func (o *ChartObserver) StopChartObserver() {
	close(o.stopChan)
}

func (o *ChartObserver) run() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Start a goroutine to listen for results
	go func() {
		for {
			select {
			case result := <-o.resultChan:
				fmt.Printf("Received result: %d\n", result)
			case <-o.stopChan:
				return
			}
		}
	}()

	for {
		select {
		case <-ticker.C:
			if rand.Intn(2) == 1 {
				result := o.functionA()
				o.resultChan <- result
			} else {
				result := o.functionB()
				o.resultChan <- result
			}
		case <-o.stopChan:
			return
		}
	}
}

func (o *ChartObserver) functionA() int {
	log.Info().Msg("Function A executed")
	return 10
}

func (o *ChartObserver) functionB() int {
	log.Info().Msg("Function B executed")
	return 20
}
