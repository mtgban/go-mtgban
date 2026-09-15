// Command tokenpairgen finds two-sided token pairings Star City Games and
// Card Trader both sell but TCGplayer's own tokenProducts feed never linked
// as one product - the candidates mtgmatcher/magic/verified_pairs_data.go
// hard-codes (see that file's own doc comment for what the table is and
// why "measure once against real vendor data, hard-code it" is the
// discipline here, same as idCanonicalKey/isMemorabiliaSet).
//
// It reuses each vendor's own real anchoring logic -
// starcitygames.TokenPairAnchorUUIDs, cardtrader.TokenPairAnchorUUIDs -
// rather than reimplementing it, and only reports a pair
// magic.TokenPairIDByUUIDs doesn't already resolve: run against a
// datastore that already has the current table loaded (any ordinary
// build does, since mintVerifiedPairs runs at load time), this only ever
// prints genuinely new candidates, never ones already on file.
//
// Prints Go source for the slice literal's own entries to stdout, one
// vendor-verified pair per line with its face names as a trailing
// comment, in the exact shape verified_pairs_data.go's own entries use -
// splice the printed lines into that file's existing literal by hand, and
// update its doc comment's own collision-count and "collected" date to
// match, rather than having this tool overwrite hand-authored prose.
//
// A printed candidate is not yet a verified one: it is only as trustworthy
// as the anchor that found it. A same-filing-set pair (both faces' own sku
// anchors naming the identical set) is the safe, common case the existing
// table is mostly made of. A pair whose two anchors name two different
// sets is real by construction - each face still resolves to exactly one
// printing - but is also the shape the sku itself can produce by bundling
// two unrelated tokens SCG chose to sell as one listing rather than by two
// faces of one physical card; treat those, and anything anchored off a
// PWSB/SECRET/other shelf-marker prefix, as needing the same one-by-one
// cross-check against the vendor's own live product page the original 664
// rows got before trusting either as a real physical pairing.
//
// Usage:
//
//	ALLPRINTINGS5_PATH=... SCG_API_KEY=... CARDTRADER_TOKEN_BEARER=... \
//	  go run ./cmd/tokenpairgen
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/mtgban/go-mtgban/cardtrader"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/magic"
	"github.com/mtgban/go-mtgban/starcitygames"
)

func orderPair(a, b string) (string, string) {
	if b < a {
		return b, a
	}
	return a, b
}

// scanSCG streams the full SCG catalog and returns every candidate pair its
// own sku anchors both faces of, keyed by ordered uuid pair.
func scanSCG(ctx context.Context, apiKey string) (map[[2]string]bool, error) {
	scg := starcitygames.NewSCGClient(apiKey)
	found := map[[2]string]bool{}
	err := scg.StreamCatalog(ctx, func() { found = map[[2]string]bool{} }, func(p starcitygames.CatalogProduct) error {
		uuidA, uuidB, ok := starcitygames.TokenPairAnchorUUIDs(p)
		if !ok || uuidA == uuidB {
			return nil
		}
		lo, hi := orderPair(uuidA, uuidB)
		found[[2]string{lo, hi}] = true
		return nil
	})
	return found, err
}

// scanCardTrader walks every Magic expansion's own blueprints and returns
// every candidate pair a blueprint's own composite collector_number anchors
// both faces of, keyed by ordered uuid pair. Fetched and formatted the same
// way the real scraper's own Load does (BlueprintsForGameID, FormatBlueprints)
// rather than calling the raw Blueprints endpoint directly - it leaves
// Properties.Number empty, the real one arriving under FixedProperties
// instead until FormatBlueprints moves it over, the same way cardtrader.go's
// own Load has to.
func scanCardTrader(ctx context.Context, token string) (map[[2]string]bool, error) {
	client := cardtrader.NewCTAuthClient(token)

	blueprintsRaw, expansionsRaw, err := cardtrader.BlueprintsForGameID(ctx, client, cardtrader.GameMagic, "", func(format string, args ...any) {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
	})
	if err != nil {
		return nil, fmt.Errorf("blueprints: %w", err)
	}
	fmt.Fprintln(os.Stderr, "cardtrader: fetched", len(blueprintsRaw), "blueprints across", len(expansionsRaw), "expansions")

	blueprints, _ := cardtrader.FormatBlueprints(blueprintsRaw, expansionsRaw, false)

	found := map[[2]string]bool{}
	for _, bp := range blueprints {
		uuidA, uuidB, ok := cardtrader.TokenPairAnchorUUIDs(*bp)
		if !ok || uuidA == uuidB {
			continue
		}
		lo, hi := orderPair(uuidA, uuidB)
		found[[2]string{lo, hi}] = true
	}
	return found, nil
}

func run() int {
	allprintingsPath := os.Getenv("ALLPRINTINGS5_PATH")
	if allprintingsPath == "" {
		allprintingsPath = "allprintings5.json"
	}
	f, err := os.Open(allprintingsPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer f.Close()
	ds, err := magic.Load(f)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	mtgmatcher.SetGlobalDatastore(ds)

	known := magic.TokenPairIDByUUIDs()
	all := map[[2]string]bool{}

	ctx := context.Background()

	if apiKey := os.Getenv("SCG_API_KEY"); apiKey != "" {
		found, err := scanSCG(ctx, apiKey)
		if err != nil {
			fmt.Fprintln(os.Stderr, "scg:", err)
			return 1
		}
		fmt.Fprintln(os.Stderr, "scg: found", len(found), "candidate pairs")
		for k := range found {
			all[k] = true
		}
	} else {
		fmt.Fprintln(os.Stderr, "SCG_API_KEY not set, skipping Star City Games")
	}

	if token := os.Getenv("CARDTRADER_TOKEN_BEARER"); token != "" {
		found, err := scanCardTrader(ctx, token)
		if err != nil {
			fmt.Fprintln(os.Stderr, "cardtrader:", err)
			return 1
		}
		fmt.Fprintln(os.Stderr, "cardtrader: found", len(found), "candidate pairs")
		for k := range found {
			all[k] = true
		}
	} else {
		fmt.Fprintln(os.Stderr, "CARDTRADER_TOKEN_BEARER not set, skipping Card Trader")
	}

	var fresh [][2]string
	for k := range all {
		if known[k] == "" {
			fresh = append(fresh, k)
		}
	}
	sort.Slice(fresh, func(i, j int) bool {
		if fresh[i][0] != fresh[j][0] {
			return fresh[i][0] < fresh[j][0]
		}
		return fresh[i][1] < fresh[j][1]
	})

	fmt.Fprintln(os.Stderr, len(fresh), "pairs are new since the table currently loaded")

	for _, k := range fresh {
		coA, errA := mtgmatcher.GetUUID(k[0])
		coB, errB := mtgmatcher.GetUUID(k[1])
		nameA, nameB := k[0], k[1]
		if errA == nil {
			nameA = coA.Name
		}
		if errB == nil {
			nameB = coB.Name
		}
		fmt.Fprintf(os.Stdout, "\t{%q, %q}, // %s // %s\n", k[0], k[1], nameA, nameB)
	}

	return 0
}

func main() {
	flag.Parse()
	os.Exit(run())
}
