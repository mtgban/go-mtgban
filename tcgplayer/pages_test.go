package tcgplayer

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/mtgban/go-tcgplayer"
)

func TestCheckPage(t *testing.T) {
	for _, tt := range []struct {
		page, total, got int
		ok               bool
	}{
		{0, 250, 100, true},
		{0, 250, 99, false},
		{200, 250, 50, true},
		{200, 250, 49, false},
		{200, 250, 51, false},
	} {
		err := checkPage(make([]tcgplayer.Product, tt.got), tt.page, tt.total)
		if (err == nil) != tt.ok {
			t.Errorf("checkPage(%d products, offset %d of %d) error = %v, want ok=%v", tt.got, tt.page, tt.total, err, tt.ok)
		}
	}
}

// WorkerPool only logs a page's error; loadPages is what turns one into a
// failed Load rather than an inventory missing that page.
func TestLoadPagesFailsWhenAPageDoes(t *testing.T) {
	var mtx sync.Mutex
	var pages []int
	var consumed int
	run := func(failAt int) error {
		pages, consumed = nil, 0
		return loadPages(context.Background(), 2, 250,
			func(ctx context.Context, page int, channel chan<- genericChan) error {
				mtx.Lock()
				pages = append(pages, page)
				mtx.Unlock()
				if page == failAt {
					return errors.New("page refused")
				}
				channel <- genericChan{key: "k"}
				return nil
			},
			func(genericChan) { consumed++ },
			func(string, ...any) {},
		)
	}

	if err := run(-1); err != nil {
		t.Errorf("loadPages() with every page good error = %v, want nil", err)
	}
	slices.Sort(pages)
	if !slices.Equal(pages, []int{0, 100, 200}) || consumed != 3 {
		t.Errorf("loadPages() visited %v and consumed %d, want [0 100 200] and 3", pages, consumed)
	}

	err := run(100)
	if err == nil || !strings.Contains(err.Error(), "1 of 3 pages") {
		t.Errorf("loadPages() with a failed page error = %v, want 1 of 3 pages reported", err)
	}
}
