# ADR-0005: The matcher package is named `matcher`

**Status:** Accepted
**Date:** 2026-10-25

## Context

The package every listing is matched through was named `mtgmatcher` when it
matched Magic alone. It now matches nine games, each in its own sub-package
behind the `GameRules` seam, and holds no Magic vocabulary of its own: the
Magic-only helpers, `IsBasicLand` and the game names moved out or became
game-neutral before this decision. The name was the last thing in the core
saying it was about one game, and every import line in the module, the site
and its tools repeated it.

## Decision

`mtgmatcher/` becomes `matcher/`, at `github.com/mtgban/go-mtgban/matcher`,
with every game package under it (`matcher/magic`, `matcher/pokemon`, ...).
The core file `mtgmatcher.go` becomes `matcher.go`. Nothing else is renamed:
the module path, the `mtgban` package (the project's name, not a game's) and
every exported identifier stay as they are.

## Alternatives

- **Keep the name.** Free, but the one Magic-sounding name in the core is the
  one every caller reads first.
- **A compatibility package forwarding the old path.** Go has no package
  alias, so it would forward every exported symbol of the core and its eleven
  sub-packages by hand, for the few consumers this project owns.
- **`catalog` or `cards`.** Both are already identifiers in the module and
  the site; `matcher` is used as one in neither.

## Consequences

- The change is mechanical: the directory move, a whole-word replace of
  `mtgmatcher` in Go and docs, and gofmt to re-sort imports. Re-running the
  replace outside `docs/adr/` repairs a branch rebased across the rename; a
  file it added under `mtgmatcher/` that git left there moves by hand.
- `git log -- matcher/` stops at the rename; `git log --follow` on a file, or
  `-- mtgmatcher/` before it, reaches older history.
- ADR-0001 to ADR-0004 keep the old name, as accepted ADRs are not edited.
- Every consumer renames its imports at its first go-mtgban bump past the
  rename; until then it stays on an older tag.

## Action items

- mtgban-website and img-downloader: rename the imports at that bump.
