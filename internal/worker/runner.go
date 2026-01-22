package worker

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Runner struct {
	poller      *Poller
	handler     Handler
	maxInFlight int
}

func NewRunner(poller *Poller, handler Handler, maxInFlight int) *Runner {
	return &Runner{
		poller:      poller,
		handler:     handler,
		maxInFlight: maxInFlight,
	}
}

func (r *Runner) Run(ctx context.Context) error {
	sem := make(chan struct{}, r.maxInFlight)
	var wg sync.WaitGroup
	defer wg.Wait()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case sem <- struct{}{}:
			// acquired slot
		}

		msg, err := r.poller.ReceiveOne(ctx)
		if err != nil {
			<-sem
			fmt.Printf("receive error: %v\n", err)
			continue
		}
		if msg == nil {
			<-sem
			continue
		}

		wg.Add(1)
		go func(m *Message) {
			defer wg.Done()
			defer func() { <-sem }()

			// Handler respects shutdown but has per-message timeout
			handlerCtx, handlerCancel := context.WithTimeout(ctx, 30*time.Second)
			defer handlerCancel()
			if err := r.handler(handlerCtx, m); err != nil {
				fmt.Printf("handler error: %v\n", err)
				return
			}

			// Delete gets grace period after shutdown
			delCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // TODO: tune
			defer cancel()

			if err := r.poller.Delete(delCtx, m); err != nil {
				fmt.Printf("delete error: %v\n", err)
			}
		}(msg)
	}
}
