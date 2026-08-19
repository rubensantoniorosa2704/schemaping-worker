package scheduler

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/rubensantoniorosa2704/schemaping-worker/internal/domain"
)

// Run fires fn(m) immediately for each monitor, then on each monitor's interval.
// Blocks until SIGINT or SIGTERM is received.
func Run(monitors []domain.Monitor, fn func(domain.Monitor)) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup
	stops := make([]chan struct{}, len(monitors))

	for i, m := range monitors {
		stops[i] = make(chan struct{})
		wg.Add(1)
		go func(m domain.Monitor, stop chan struct{}) {
			defer wg.Done()
			fn(m)
			ticker := time.NewTicker(m.Interval)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					fn(m)
				case <-stop:
					return
				}
			}
		}(m, stops[i])
	}

	<-quit
	signal.Stop(quit)

	for _, stop := range stops {
		close(stop)
	}
	wg.Wait()
}
