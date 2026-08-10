package mtgban

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"
)

// collect runs WorkerPool in the background and fails if it has not returned
// before the deadline. A pool that never returns would otherwise hang the
// whole test binary rather than reporting which case broke.
func collect(t *testing.T, concurrency int, items []int) []int {
	t.Helper()

	var mu sync.Mutex
	var got []int

	done := make(chan struct{})
	go func() {
		defer close(done)
		WorkerPool(context.Background(), concurrency, items,
			func(ctx context.Context, item int, results chan<- int) error {
				results <- item * 2
				return nil
			},
			func(result int) {
				mu.Lock()
				defer mu.Unlock()
				got = append(got, result)
			},
			nil,
		)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatalf("WorkerPool did not return with concurrency %d", concurrency)
	}

	slices.Sort(got)
	return got
}

func TestWorkerPool(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}
	want := []int{2, 4, 6, 8, 10}

	// Zero is what a scraper that forgot to set MaxConcurrency passes in.
	for _, concurrency := range []int{-1, 0, 1, 3, 100} {
		got := collect(t, concurrency, items)
		if !slices.Equal(got, want) {
			t.Errorf("concurrency %d: got %v, want %v", concurrency, got, want)
		}
	}
}

func TestWorkerPoolNoItems(t *testing.T) {
	got := collect(t, 0, nil)
	if len(got) != 0 {
		t.Errorf("got %v, want nothing", got)
	}
}

func TestWorkerPoolCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		WorkerPool(ctx, 2, []int{1, 2, 3},
			func(ctx context.Context, item int, results chan<- int) error {
				results <- item
				return nil
			},
			func(int) {},
			nil,
		)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("WorkerPool did not return for a cancelled context")
	}
}

func TestWorkerPoolReportsWorkerErrors(t *testing.T) {
	var mu sync.Mutex
	var logged int

	WorkerPool(context.Background(), 2, []int{1, 2, 3},
		func(ctx context.Context, item int, results chan<- int) error {
			return context.DeadlineExceeded
		},
		func(int) {},
		func(format string, a ...interface{}) {
			mu.Lock()
			defer mu.Unlock()
			logged++
		},
	)

	if logged != 3 {
		t.Errorf("logged %d errors, want 3", logged)
	}
}
