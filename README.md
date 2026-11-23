# go-mtgban

[![CI](https://github.com/mtgban/go-mtgban/actions/workflows/ci.yml/badge.svg)](https://github.com/mtgban/go-mtgban/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/mtgban/go-mtgban.svg)](https://pkg.go.dev/github.com/mtgban/go-mtgban)
[![License: AGPL-3.0 or Commercial](https://img.shields.io/badge/license-AGPL--3.0%20%7C%20Commercial-blue)](LICENSING.md)

go-mtgban is the market-data engine behind [BAN](https://www.mtgban.com).
It scrapes the retail inventories and buylists of card stores and
marketplaces, resolves every listing to one canonical identity, and
compares prices across stores to find arbitrage. That covers single cards in
every condition, graded cards, and sealed product.

Most of the work goes into solving identity: deciding which exact printing a
store means when no two stores describe it the same way.

Nine trading-card games are supported: Magic: The Gathering, Disney Lorcana,
Riftbound, One Piece, Yu-Gi-Oh, Flesh and Blood, Pokemon, Gundam, and
Palworld. Over 120 scheduled runs, each one scraper for one game, refresh
prices at least once a day, and more than 2,000 recorded listings pin down
how cards match.

The project is in active development, with
[tagged releases](https://github.com/mtgban/go-mtgban/tags). It is still v0,
so the Go API can change between them.

## Licensing

go-mtgban is **dual licensed**:

- **[AGPL-3.0](LICENSE)** for open source use. If you run a modified
  version as a network service, the AGPL requires you to offer the users
  of that service its complete corresponding source, including the
  components needed to build and run it.
- **Commercial license** for anyone who cannot or does not want to meet
  the AGPL's obligations. It covers the same source code under
  permissive-style terms, comes with support and integration help, and
  funds the project.

```
SPDX-License-Identifier: LicenseRef-BAN-Commercial OR AGPL-3.0
```

Contributions fall under a Contributor License Agreement, which lets
BAN Inc offer the commercial license without taking ownership of your
copyright, and contributors get a discount on the commercial license.
BAN commits to always keeping an AGPL version available.
[LICENSING.md](LICENSING.md) has the full terms.

## The two core modules

Everything else in the repository, one package per store plus the tools
under `cmd/`, is built on these two packages.

| Package | What it solves | Reference |
| --- | --- | --- |
| [`mtgban`](mtgban) | What a store publishes, and how prices compare | [docs](https://pkg.go.dev/github.com/mtgban/go-mtgban/mtgban) |
| [`matcher`](matcher) | Which card a listing means, with one rule set per game | [docs](https://pkg.go.dev/github.com/mtgban/go-mtgban/matcher) |
| [`matcher/sealed`](matcher/sealed) | What a sealed product contains | [docs](https://pkg.go.dev/github.com/mtgban/go-mtgban/matcher/sealed) |
| Store packages | Translating one store's listings each | one directory per store |
| [`cmd/bantool`](cmd/bantool) | Running the scrapers and writing their results | [workflows](.github/workflows/README.md) |

Built with:

- **Language**: Go
- **Libraries**: `goquery` for HTML and `go-retryablehttp` for requests
- **Data**: [MTGJSON](https://mtgjson.com) for Magic, and
  [datastore-gen](https://github.com/mtgban/datastore-gen) for the other games
- **Infrastructure**: GitHub Actions for scheduling, Backblaze B2 for storage

### `matcher`: one identity for every card

Stores never agree on how to name a card. One calls it
`Lightning Bolt (Borderless)`, another lists it under a promo set with a
different name, a third gets the foil flag wrong. `matcher`
resolves all of them to the same **unique identifier**, and every price in
the system is keyed on it.

- `matcher` is the game-agnostic core: the `Backend` that holds a loaded
  datastore, the shared `Match` pipeline, and the search API.
- `matcher/<game>` holds one loader and one `GameRules` implementation per
  game: the editions, numbers, promos, and finishes that game spells its own
  way.
- Games register themselves, so a program only links the ones it imports
  (or all nine through `matcher/games`).

```go
import (
	"fmt"
	"os"

	"github.com/mtgban/go-mtgban/matcher"
	_ "github.com/mtgban/go-mtgban/matcher/magic"
)

datastore, _ := os.Open("AllPrintings.json")
backend, err := matcher.Open(matcher.GameMagic, datastore)
if err != nil {
	return err
}

cardID, err := backend.Match(&matcher.InputCard{
	Name:    "Lightning Bolt",
	Edition: "Magic 2010",
})
if err != nil {
	return err
}
card, _ := backend.GetUUID(cardID)
fmt.Println(card.Name, card.SetCode, card.Number) // Lightning Bolt M10 146
```

When a card is mismatched, there may be more than one place the fix could
live. A rule that applies across stores belongs in `matcher`; something
particular to one store, such as a typo in its listings, belongs in that
store's scraper.

### `mtgban`: scrapers, records, and analysis

`mtgban` defines what a store is and what it publishes:

- **Interfaces**: a `Seller` publishes an `InventoryRecord` (what a store
  sells), a `Vendor` publishes a `BuylistRecord` (what a store pays). A
  marketplace that does both implements both.
- **Records**: price entries keyed by that identifier, inserted through the
  `Add*` helpers that validate conditions, prices, and quantities and keep
  entries sorted.
- **Registry**: each store package registers itself for the games it
  prices; `mtgban.NewScraper(backend, name, opts...)` builds one against a
  datastore.
- **Analysis**: `Arbit` finds cards one store sells for less than another
  buys them, and `Mismatch` compares two sellers' prices.
- **I/O and plumbing**: CSV and JSON writers, the shared HTTP client, and
  the `WorkerPool` every scraper fetches with.

```go
scraper, err := mtgban.NewScraper(backend, "cardkingdom")
if err != nil {
	return err
}
err = scraper.Load(ctx)
if err != nil {
	return err
}
inventory := scraper.(mtgban.Seller).Inventory()
```

In production, `cmd/bantool` runs every scraper on a schedule through
GitHub Actions:

```sh
mkdir -p out
go run ./cmd/bantool -game magic -store cardkingdom \
	-datastore AllPrintings.json -output-path out
```

### Product types

The same pipeline covers **singles**, **graded cards**, and **sealed
product**, all keyed by a unique identifier.

Singles have retail and buylist prices for each printing and finish, by
condition and with the quantity where it is known. Graded cards have retail
prices, kept apart from those of raw copies, with the grading service's
score read as a condition. Sealed product has retail and buylist prices for
boxes, packs, decks, and bundles. For Magic, it also has the contents of a
product and the expected value of opening it.

## Contributing

Contributions are welcome, and are covered by the
[Contributor License Agreement](LICENSING.md#contributors). In short:

1. Branch off `master`, and keep each pull request to a single fix or
   feature.
2. Run `gofmt -s -l .`, `go vet ./...`, `go build ./...`, and
   `go test ./...` before pushing, along with the linters CI runs.
3. Write commits as `area: imperative summary`.

[CONTRIBUTING.md](CONTRIBUTING.md) has the full guide: the datastores the
tests need, every CI check, the golden suites, and the house rules.
