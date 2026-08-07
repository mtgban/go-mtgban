package mtgban

import (
	"context"
	"errors"
	"testing"
)

func TestWorkerPoolCountsFailures(t *testing.T) {
	items := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}

	// Every odd item fails, so half the run is lost while consume still sees
	// the other half: the case the count exists to expose
	var consumed int
	failed := WorkerPool(context.Background(), 4, items,
		func(ctx context.Context, item int, results chan<- int) error {
			if item%2 == 1 {
				return errors.New("nope")
			}
			results <- item
			return nil
		},
		func(int) { consumed++ },
		nil,
	)

	if failed != 5 {
		t.Errorf("counted %d failures, expected 5", failed)
	}
	if consumed != 5 {
		t.Errorf("consumed %d results, expected 5", consumed)
	}
}

func TestWorkerPoolCountsEverythingOnSuccess(t *testing.T) {
	failed := WorkerPool(context.Background(), 4, []int{1, 2, 3},
		func(ctx context.Context, item int, results chan<- int) error {
			results <- item
			return nil
		},
		func(int) {},
		nil,
	)

	if failed != 0 {
		t.Errorf("counted %d failures on a clean run, expected 0", failed)
	}
}

// Items dropped because the caller gave up have not been processed either,
// and a run that stopped early is not a complete one.
func TestWorkerPoolCountsUndispatchedItems(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	items := make([]int, 100)
	failed := WorkerPool(ctx, 2, items,
		func(ctx context.Context, item int, results chan<- int) error {
			results <- item
			return nil
		},
		func(int) {},
		nil,
	)

	if failed == 0 {
		t.Error("a cancelled run reported no incomplete items")
	}
	if failed > len(items) {
		t.Errorf("counted %d incomplete items, more than the %d dispatched", failed, len(items))
	}
}
