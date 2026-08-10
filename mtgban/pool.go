package mtgban

import (
	"context"
	"sync"
)

// WorkerPool runs worker on each item from items with bounded concurrency.
// Each worker receives a channel to push results incrementally.
// Results are consumed on the calling goroutine via the consume callback.
// When ctx is cancelled, no new items are dispatched and in-flight workers
// are allowed to finish so that partial results are still consumed.
// A concurrency below one is raised to one rather than left to hang.
func WorkerPool[T any, R any](
	ctx context.Context,
	concurrency int,
	items []T,
	worker func(context.Context, T, chan<- R) error,
	consume func(R),
	logErr func(string, ...interface{}),
) {
	// Without this, a caller that never set its concurrency spawns no workers
	// at all: nothing drains work, so the dispatch loop blocks on its first
	// send, results is never closed, and the range below waits on it forever.
	if concurrency < 1 {
		concurrency = 1
	}

	work := make(chan T)
	results := make(chan R)
	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range work {
				err := worker(ctx, item, results)
				if err != nil && logErr != nil {
					logErr("%v", err)
				}
			}
		}()
	}

	go func() {
		for _, item := range items {
			select {
			case work <- item:
			case <-ctx.Done():
				// Stop dispatching new work
				goto done
			}
		}
	done:
		close(work)
		wg.Wait()
		close(results)
	}()

	for result := range results {
		consume(result)
	}
}
