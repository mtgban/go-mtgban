package main

import (
	"context"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgban"
)

// plainScraper holds both halves and cannot be told to drop one, which is the
// shape six of the registered packages are in.
type plainScraper struct{}

func (plainScraper) Load(context.Context) error { return nil }
func (plainScraper) Info() mtgban.ScraperInfo   { return mtgban.ScraperInfo{Name: "plain"} }

// configurableScraper answers the option, and records what it was told so the
// test can check the halves are not swapped on the way through.
type configurableScraper struct{ got mtgban.ScraperOptions }

func (*configurableScraper) Load(context.Context) error { return nil }
func (*configurableScraper) Info() mtgban.ScraperInfo {
	return mtgban.ScraperInfo{Name: "configurable"}
}
func (c *configurableScraper) SetConfig(opt mtgban.ScraperOptions) {
	c.got = opt
}

// TestConfigureScraperRefusesWhatItCannotHonour pins that asking for one half
// of a scraper that publishes both is an error rather than a silent whole run.
// The option reaches a scraper through a type assertion that says nothing when
// it fails, so this is the only place the mismatch can be named.
func TestConfigureScraperRefusesWhatItCannotHonour(t *testing.T) {
	for _, tt := range []struct {
		desc     string
		opt      scraperOption
		wantErr  bool
		wantHalf string
	}{
		{"asked for the buylist alone", scraperOption{OnlyVendor: true}, true, "buylist"},
		{"asked for the retail alone", scraperOption{OnlySeller: true}, true, "retail"},
		// A scraper with one half of its own is registered without either
		// option, and there is nothing to honour: most targets are these.
		{"asked for neither", scraperOption{}, false, ""},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			err := configureScraper("sometarget", &tt.opt, plainScraper{})
			if !tt.wantErr {
				if err != nil {
					t.Fatalf("got %v, want no error", err)
				}
				return
			}
			if err == nil {
				t.Fatal("got no error, want one naming the target")
			}
			if !strings.Contains(err.Error(), "sometarget") {
				t.Errorf("error does not name the target: %v", err)
			}
			if !strings.Contains(err.Error(), tt.wantHalf) {
				t.Errorf("error does not say which half was asked for: %v", err)
			}
		})
	}
}

// TestConfigureScraperPassesTheOptionThrough pins the other side: a scraper
// that answers the option is told exactly what was asked, and the two halves
// are not crossed - OnlyVendor turns retail off, not the buylist it names.
func TestConfigureScraperPassesTheOptionThrough(t *testing.T) {
	for _, tt := range []struct {
		desc string
		opt  scraperOption
		want mtgban.ScraperOptions
	}{
		{"the buylist alone drops retail", scraperOption{OnlyVendor: true},
			mtgban.ScraperOptions{DisableRetail: true}},
		{"the retail alone drops the buylist", scraperOption{OnlySeller: true},
			mtgban.ScraperOptions{DisableBuylist: true}},
		{"neither drops nothing", scraperOption{}, mtgban.ScraperOptions{}},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			scraper := &configurableScraper{}
			err := configureScraper("sometarget", &tt.opt, scraper)
			if err != nil {
				t.Fatal(err)
			}
			if scraper.got != tt.want {
				t.Errorf("scraper was told %+v, want %+v", scraper.got, tt.want)
			}
		})
	}
}

// TestRegisteredHalvesAreHonoured walks the targets that ask for one half and
// checks each can actually be told to drop the other. Registering the option
// on a scraper that cannot answer it is silent at runtime and shows up only as
// an output holding the half that was not asked for, which is how Vegas
// Singles published an empty Magic shelf twice a day.
func TestRegisteredHalvesAreHonoured(t *testing.T) {
	var checked int
	for name, opt := range options {
		if !opt.OnlyVendor && !opt.OnlySeller {
			continue
		}
		scraper, err := opt.Init()
		if err != nil {
			// Init reads credentials for some targets, and a checkout
			// without them still runs the rest.
			t.Logf("skipping %s: %v", name, err)
			continue
		}
		checked++
		if err := configureScraper(name, opt, scraper); err != nil {
			t.Error(err)
		}
	}
	if checked == 0 {
		t.Skip("no target asks for a single half, or none could be built here")
	}
	t.Logf("checked %d targets that ask for a single half", checked)
}
