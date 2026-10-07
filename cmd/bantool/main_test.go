package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/simplecloud"
)

// An empty scrape must count as no data even when the scraper reported a
// timestamp and unfolded into named sellers and vendors: those come back
// with empty records, and uploading them would blank the bucket.
func TestCountResults(t *testing.T) {
	var info mtgban.ScraperInfo
	emptySeller := mtgban.NewSellerFromInventory(nil, info)
	emptyVendor := mtgban.NewVendorFromBuylist(nil, info)
	fullSeller := mtgban.NewSellerFromInventory(mtgban.InventoryRecord{
		"uuid": {{Price: 1}},
	}, info)
	fullVendor := mtgban.NewVendorFromBuylist(mtgban.BuylistRecord{
		"uuid": {{BuyPrice: 1}},
	}, info)

	tests := []struct {
		name        string
		sellers     []mtgban.Seller
		vendors     []mtgban.Vendor
		wantRetail  int
		wantBuylist int
	}{
		{"nothing unfolded at all", nil, nil, 0, 0},
		{"a seller with an empty record", []mtgban.Seller{emptySeller}, nil, 0, 0},
		{"empty on both sides", []mtgban.Seller{emptySeller}, []mtgban.Vendor{emptyVendor}, 0, 0},
		{"one priced card counts", []mtgban.Seller{fullSeller}, []mtgban.Vendor{emptyVendor}, 1, 0},
		{"a buylist card counts apart", []mtgban.Seller{emptySeller}, []mtgban.Vendor{fullVendor}, 0, 1},
		{"two sellers of the same card both count", []mtgban.Seller{fullSeller, fullSeller}, []mtgban.Vendor{fullVendor}, 2, 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			retail, buylist := countResults(test.sellers, test.vendors)
			if retail != test.wantRetail || buylist != test.wantBuylist {
				t.Errorf("countResults() = (%d, %d), want (%d, %d)",
					retail, buylist, test.wantRetail, test.wantBuylist)
			}
		})
	}
}

// commitBucket stands in for a cloud bucket, where Close is what publishes an
// object and Abort discards it.
type commitBucket struct {
	panics    bool
	committed bool
	aborted   bool
}

func (b *commitBucket) NewWriter(context.Context, string) (io.WriteCloser, error) {
	return commitWriter{b}, nil
}

type commitWriter struct {
	bucket *commitBucket
}

// writePanic is what a commitWriter panics with on its first Write when its
// bucket asks it to.
const writePanic = "the storage stream broke"

func (w commitWriter) Write(p []byte) (int, error) {
	if w.bucket.panics {
		panic(writePanic)
	}
	return len(p), nil
}

func (w commitWriter) Close() error {
	w.bucket.committed = true
	return nil
}

func (w commitWriter) Abort() error {
	w.bucket.aborted = true
	return nil
}

// dumpHalves dumps a one-card record through dumpSeller or dumpVendor, so the
// tests below hold both to the same rule.
var dumpHalves = []struct {
	name string
	dump func(bucket simplecloud.Writer, format string) error
}{
	{"seller", func(bucket simplecloud.Writer, format string) error {
		inventory := mtgban.InventoryRecord{"uuid": {{Price: 1}}}
		seller := mtgban.NewSellerFromInventory(inventory, mtgban.ScraperInfo{Shorthand: "TEST"})
		return dumpSeller(nil, bucket, seller, "out", format)
	}},
	{"vendor", func(bucket simplecloud.Writer, format string) error {
		buylist := mtgban.BuylistRecord{"uuid": {{BuyPrice: 1}}}
		vendor := mtgban.NewVendorFromBuylist(buylist, mtgban.ScraperInfo{Shorthand: "TEST"})
		return dumpVendor(nil, bucket, vendor, "out", format)
	}},
}

// A dump whose encode fails must be discarded, never closed: Close is what
// publishes on the cloud backends, so closing would replace the store's last
// good dump with an empty or truncated one.
func TestDumpAbortsAFailedEncode(t *testing.T) {
	for _, half := range dumpHalves {
		t.Run(half.name, func(t *testing.T) {
			bucket := &commitBucket{}
			err := half.dump(bucket, "xml")
			if err == nil {
				t.Error("an unknown format was dumped without error")
			}
			if bucket.committed || !bucket.aborted {
				t.Errorf("committed %v, aborted %v; want an abort alone", bucket.committed, bucket.aborted)
			}
		})
	}
}

// A panic mid-encode must abort the write as it unwinds, for the same reason,
// and still reach the caller.
func TestDumpAbortsOnPanic(t *testing.T) {
	for _, half := range dumpHalves {
		t.Run(half.name, func(t *testing.T) {
			bucket := &commitBucket{panics: true}
			defer func() {
				r := recover()
				if r != writePanic {
					t.Errorf("recovered %v, want the writer's own panic", r)
				}
				if bucket.committed || !bucket.aborted {
					t.Errorf("committed %v, aborted %v; want an abort alone", bucket.committed, bucket.aborted)
				}
			}()
			err := half.dump(bucket, "json")
			t.Errorf("dump returned %v instead of panicking", err)
		})
	}
}

// A complete encode, in the format the workflows publish, is committed and
// never aborted.
func TestDumpCommitsACompleteEncode(t *testing.T) {
	for _, half := range dumpHalves {
		t.Run(half.name, func(t *testing.T) {
			bucket := &commitBucket{}
			err := half.dump(bucket, "json.xz")
			if err != nil {
				t.Error(err)
			}
			if !bucket.committed || bucket.aborted {
				t.Errorf("committed %v, aborted %v; want a commit alone", bucket.committed, bucket.aborted)
			}
		})
	}
}

// loadStub is a scraper that holds a row whether or not its Load fails.
type loadStub struct {
	name string
	err  error
}

func (s *loadStub) Load(context.Context) error { return s.err }

func (s *loadStub) Info() mtgban.ScraperInfo {
	return mtgban.ScraperInfo{Shorthand: s.name}
}

func (s *loadStub) Inventory() mtgban.InventoryRecord {
	return mtgban.InventoryRecord{"uuid": {{Price: 1}}}
}

// bothStub is a loadStub that also buys, loading both halves in one Load.
type bothStub struct{ loadStub }

func (s *bothStub) Buylist() mtgban.BuylistRecord {
	return mtgban.BuylistRecord{"uuid": {{BuyPrice: 1}}}
}

// A scraper whose Load failed may hold part of its data, which must not
// reach the dump, while the scrapers around it, and the half of a scraper
// that loaded when its error names the other half alone, still do.
func TestLoadKeepsOnlyWhatLoaded(t *testing.T) {
	retailErr := fmt.Errorf("%w: %w", mtgban.ErrInventoryLoad, errors.New("timeout"))
	buylistErr := fmt.Errorf("%w: %w", mtgban.ErrBuylistLoad, errors.New("timeout"))
	scrapers := []mtgban.Scraper{
		&bothStub{loadStub{name: "A"}},
		&bothStub{loadStub{name: "B", err: errors.Join(buylistErr)}},
		&bothStub{loadStub{name: "C", err: errors.Join(retailErr, buylistErr)}},
		&loadStub{name: "D", err: errors.New("2 of 3 pages failed")},
		&bothStub{loadStub{name: "E", err: errors.Join(retailErr)}},
		&loadStub{name: "F"},
	}
	sellers, vendors, errs := load(context.Background(), scrapers)

	var got []string
	for _, seller := range sellers {
		got = append(got, "sell "+seller.Info().Shorthand)
	}
	for _, vendor := range vendors {
		got = append(got, "buy "+vendor.Info().Shorthand)
	}
	want := []string{"sell A", "sell B", "sell F", "buy A", "buy E"}
	if !slices.Equal(got, want) {
		t.Errorf("load() kept %q, want %q", got, want)
	}

	got = nil
	for _, err := range errs {
		got = append(got, strings.SplitN(err.Error(), ":", 2)[0])
	}
	want = []string{"B", "C not dumped", "D not dumped", "E"}
	if !slices.Equal(got, want) {
		t.Errorf("load() errors %q, want %q", got, want)
	}
}
