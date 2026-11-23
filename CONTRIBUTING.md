# Contributing

Contributions are welcome, and are covered by the
[Contributor License Agreement](LICENSING.md#contributors).

1. **Read the docs first.** [SPECIFICATIONS.md](SPECIFICATIONS.md) covers
   the architecture, [docs/adr/](docs/adr/) the reasoning behind the big
   decisions, and [AGENTS.md](AGENTS.md) the conventions in detail.
2. **Branch off `master`**, and keep each pull request to a single fix or
   feature.
3. **Get the datastores** you need. Tests that read real card data skip
   unless their datastore is set, so a bare `go test ./...` passes with
   less coverage. Export absolute paths, for example:

   ```sh
   export ALLPRINTINGS5_PATH=/abs/path/AllPrintings.json    # Magic
   export LORCANA_PATH=/abs/path/lorcana.json               # and so on per game
   ```

4. **Run the full gate** before pushing. CI runs the same checks and fails
   on any of them:

   ```sh
   gofmt -s -l .        # must print nothing
   go vet ./...
   go run github.com/mgechev/revive@v1.17.1 -set_exit_status -config .revive.toml ./...
   go run honnef.co/go/tools/cmd/staticcheck@2026.2.2 ./...
   go build ./...
   go test ./...
   ```

5. **Respect the golden suites.** Each game keeps a replay corpus of
   listings and their expected matches under `matcher/<game>/testdata/`.
   Regenerate it only after an intentional matching change, and review the
   diff line by line. A refactor must leave the Magic corpus byte-identical.
6. **Write commits as `area: imperative summary`**, for example
   `matcher/magic: alias the new promo set`, wrapped at 80 columns.

A few house rules worth knowing up front:

- Put a matching fix where it fits: in `matcher` when it affects several
  stores, in the scraper when it is specific to one, like a store's typo.
- Insert prices through the `Add*` family, never by writing to a record's
  map directly.
- Fetch concurrently through `mtgban.WorkerPool`.
- Log through the scraper's `LogCallback`, never a global logger.
- No `reflect`, tests included.

New scrapers and new games each have a step-by-step checklist in
[AGENTS.md](AGENTS.md#adding-a-scraper).
