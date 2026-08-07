package mtgban

import (
	"context"
	"sync"
	"sync/atomic"
)

// WorkerPool runs worker on each item from items with bounded concurrency.
// Each worker receives a channel to push results incrementally.
// Results are consumed on the calling goroutine via the consume callback.
// When ctx is cancelled, no new items are dispatched and in-flight workers
// are allowed to finish so that partial results are still consumed.
//
// It returns how many items did not complete: those whose worker reported an
// error, plus those never dispatched because ctx was cancelled. Callers need
// that number to tell a whole run from a fragment of one, since consume is
// handed whatever did arrive either way.
func WorkerPool[T any, R any](
	ctx context.Context,
	concurrency int,
	items []T,
	worker func(context.Context, T, chan<- R) error,
	consume func(R),
	logErr func(string, ...interface{}),
) int {
	work := make(chan T)
	results := make(chan R)
	var wg sync.WaitGroup
	var failed atomic.Int64

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range work {
				err := worker(ctx, item, results)
				if err != nil {
					failed.Add(1)
					if logErr != nil {
						logErr("%v", err)
					}
				}
			}
		}()
	}

	go func() {
		for i, item := range items {
			select {
			case work <- item:
			case <-ctx.Done():
				// Stop dispatching: this item and every one after it
				// never ran
				failed.Add(int64(len(items) - i))
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

	// Every worker has finished by the time results is closed
	return int(failed.Load())
}
