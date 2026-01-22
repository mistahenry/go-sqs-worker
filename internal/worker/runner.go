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
	concurrency int
}

func NewRunner(poller *Poller, handler Handler, maxInFlight int, concurrency int) *Runner {
	return &Runner{
		poller:      poller,
		handler:     handler,
		maxInFlight: maxInFlight,
		concurrency: concurrency,
	}
}

func (r *Runner) Run(ctx context.Context) error {
	// allow for buffering all messages at `maxInFlight` that don't have a worker available
	messageBufferSize := r.maxInFlight - r.concurrency
	if messageBufferSize < 0 {
		messageBufferSize = 0 // unbuffered if workers >= maxInFlight
	}
	msgCh := make(chan *Message, messageBufferSize)
	sem := make(chan struct{}, r.maxInFlight)
	var wg sync.WaitGroup

	// Start worker pool based on concurrency
	for i := 0; i < r.concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			r.worker(ctx, msgCh, sem, workerID)
		}(i)
	}

	go func() {
		defer close(msgCh)
		for {
			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}: // acquire slot before receive
			}

			msg, err := r.poller.ReceiveOne(ctx)
			if err != nil {
				<-sem // release on error
				if ctx.Err() != nil {
					return
				}
				fmt.Printf("receive error: %v\n", err)
				continue
			}
			if msg == nil {
				<-sem // release if no message
				continue
			}

			select {
			case msgCh <- msg:
			case <-ctx.Done():
				<-sem
				return
			}
		}
	}()

	wg.Wait()
	return ctx.Err()
}

func (r *Runner) worker(ctx context.Context, msgCh <-chan *Message, sem <-chan struct{}, workerID int) {
	for msg := range msgCh {
		handlerCtx, cancel := context.WithTimeout(ctx, 30*time.Second)

		if err := r.handler(handlerCtx, msg); err != nil {
			cancel()
			<-sem
			fmt.Printf("worker %d handler error: %v\n", workerID, err)
			continue
		}
		cancel()

		delCtx, delCancel := context.WithTimeout(context.Background(), 2*time.Second)
		if err := r.poller.Delete(delCtx, msg); err != nil {
			fmt.Printf("worker %d delete error: %v\n", workerID, err)
		}
		delCancel()

		<-sem
	}
}
