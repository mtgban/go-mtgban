# ADR-0004: No global state in mtgmatcher; scrapers build on the datastore they are handed

**Status:** Accepted
**Date:** 2026-09-15
**Supersedes:** ADR-0002 and ADR-0003 (the global backend and its publication)

## Context

`mtgmatcher` kept one process-wide datastore behind `SetGlobalDatastore`, read
by 45 package-level wrapper functions (`GetUUID`, `Match`, `SearchEquals`, ...)
and a package `Logger`. Every scraper, the `mtgban` records and reports, the
tools and the website matched through the wrappers: roughly 460 call sites in
the library, 240 more in tests, and every scraper test began by installing a
global for the package under test and restoring the previous one after.

That shape hid what a function depended on. `InputCard.String` looked at a
card and resolved its id through the global; a scraper's constructor took a
game and then matched against whatever datastore happened to be installed,
which is
how a Magic run once matched Pokemon listings against AllPrintings. Two games
could not be priced in one process, a test could not build two backends and
compare them without swapping the world, and the publication machinery of
ADR-0003 existed only to make swapping the world survivable.

bantool composed 126 scrapers by hand in a table keyed by game and name, each
entry reading its own environment variables for credentials, partner codes
and file paths, and setting fields on the concrete scraper type. There was no
way to ask the library for "the cardmarket scraper for this datastore".

## Decision

**The datastore is a value that is passed.** `Backend` carries `Game` (the
name `Open` loaded it under) and `Logger` (nil is quiet; `Logf` and `Log`
write through it, and a game's rules write through the backend they are
handed). Every former wrapper is a method on `*Backend`; the wrappers, the
global pointer, `SetGlobalDatastore`, `GlobalDatastore`, `SetGlobalLogger` and
the package `Logger` are gone. `ExtractNumber` and `ExtractNumberAny` stay
package-level functions: the standalone `sealed-on-backend` work dropped the
skip that read the datastore to recognise a set code as something other than
a number, which is what let them stay pure through this change too.
`InputCard.String` prints the card it was given and no longer resolves an
id; `InputCard.IsGenericPromo` is gone in
favour of `Backend.IsGenericPromo`.

**A scraper holds the backend it was built on.** Every constructor takes
`b *mtgmatcher.Backend` first and no longer takes a game: the game is the
datastore's, read with `mtgban.GameOf`, so a scraper cannot be told one game
and matched against another. Free functions that matched take the backend as
a parameter. No package holds a backend in a variable.

**Scrapers register themselves on import.** `mtgban.Register(name, games,
constructor)` is called from a scraper package's `init`, under the names
bantool has always enabled scrapers by (`cardmarket`, `tcg_market`,
`sealed_ev`) and for the games each prices. A caller builds one with

    scraper, err := mtgban.NewScraper(backend, "cardmarket",
        mtgban.WithLogCallback(log.Printf),
        mtgban.WithAffiliate(partner),
        cardmarket.WithCatalog(catalog),
    )

and gets an initialized scraper to call `Load` on. `Options` carries what
every scraper understands (log callback, concurrency cap, affiliate code,
target edition, one half of a store) behind an `Option` interface only
`mtgban` implements; `WithResource` is the one door for the typed inputs a
single scraper reads (a catalog, a SKU list, an id bridge), which the scraper
package wraps in a typed option and reads back with `Resource`, refusing a
value of the wrong type at construction. `Authenticator` hands over secrets
by name; the names are the environment variables that always carried them,
so a caller reading the environment passes them through unchanged and a test
hands `MapAuthenticator` the same names. `mtgban` reads no environment of its
own: the one implementation over `os.Getenv` is bantool's, unexported, handed
in with `WithAuthenticator`. It is itself an option, since most scrapers need
no secret at all: a constructor asks `Options.Secret`, which answers the same
`ErrMissingSecret` whether the caller gave no authenticator or one that lacks
the secret, and a scraper that works without a secret ignores that error
(`cardmarket_market` prices without its BAN key).

**What stays package-level, and why.** `Normalize`'s memo cache, the regex
caches in the Gundam and Palworld rules and the `sync.OnceValue` tables are
memoization of pure functions of constants: no call's answer depends on
anything but its arguments. The one that was not - the Magic token-pairing
index, memoized over whatever datastore happened to be installed first - is
a `Backend` field now, built by the loader that minted the pairings. The game registry (`RegisterGame`) and the
scraper registry (`Register`) are init-time tables that are complete before
`main` runs and never change after. Both are disclosed here so nobody has
to rediscover that they are not the global this ADR removes.

**bantool builds from the registry.** Its flag table is derived from
`Registered(game)` for every game, with a small override table for the
targets that publish one half (`mtgseattle`, `vegassingles` on Magic,
`coolstuffinc_sealed` on Yu-Gi-Oh); the
per-target closures are replaced by `NewScraper` plus what stays bantool's:
reading credentials from the environment, loading catalogs and SKU lists
from paths, building the cardtrader bridge and deciding whether a failure to
build it is fatal (`cardmarket.BridgeUse`).

**Six scrapers convert but stay unregistered.** `cardsphere`, `mtgstocks`,
`ninetyfive`, `secretdeskorrigans`, `toamagic` and `wizardscupboard` all
took a `*Backend` first like every other package, but none gets a
`register.go`: none of the six had a bantool target before this series
either, and registering one now would demand a `bantool-<store>.yml` and a
`game` input for a target that has never been scheduled and has nothing to
schedule it. They are library-only on purpose, not half-migrated:
`TestEveryTargetIsScheduledByItsOwnWorkflow` checks that a registered
target has a workflow, never that every scraper package must register
one, so leaving these six out asks nothing of it. `ninetyfive` is further
than merely never having run under bantool — it is a store the maintainer
has retired from bantool work outright.

**mtgban's own readers take the backend.** The CSV loaders and writers and
`Pennystock` take it as their first parameter; `ArbitOpts.Backend` is the
only datastore `Arbit` and `Mismatch` use, and a nil one resolves against an
empty datastore, which is what an unpublished global did. The duplicate-entry
error names the card by id alone.

## Alternatives

| Option | Benefit | Cost |
|---|---|---|
| Keep the global, add per-game globals | Smallest diff | Still hidden dependencies; still one datastore per game per process; tests still swap the world |
| Pass the game to `NewScraper` beside the datastore | Matches the first sketch of the API | Two arguments that can disagree, for one fact the datastore already carries |
| Typed resource fields on `Options` | No `any` | `tcgplayer` and `cardmarket` types cannot be named from `mtgban` without an import cycle |
| Scrapers open their own catalogs from paths | bantool shrinks further | Vendor composition (the cardtrader bridge) and B2 credentials move into scraper packages that should know nothing of either |

## Consequences

Two datastores can live in one process and be told apart; a test builds the
backend it needs and hands it over, with nothing to restore. A datastore
reload is the caller's: build a new backend, hand it to new scrapers. The
race-detector regressions for concurrent publication go with the publication.

The website matches through the wrappers at about 500 sites and installs the
global once at startup; on the next go-mtgban bump it holds the backend it
loads and calls methods on it, passes it in `ArbitOpts` and to
`WriteBuylistToCSV`. It pins a released version, so nothing changes for it
until then.

## Action items

- [x] `Backend.Game`, `Backend.Logger`, `Logf`/`Log`, the missing methods.
- [x] `mtgban.Register`, `NewScraper`, `Options`, `Authenticator`.
- [x] Every scraper package converted and registered; bantool built from
      the registry.
- [x] The global, the wrappers and the package logger removed; docs updated.
- [ ] mtgban-website: hold the backend (tracked on that repository).
