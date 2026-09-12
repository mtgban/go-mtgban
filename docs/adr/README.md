# Architecture Decision Records

Load-bearing decisions for go-mtgban, extracted from `SPECIFICATIONS.md` so the
*reasoning* — context, alternatives, consequences — lives somewhere durable and
reviewable. An ADR is immutable once **Accepted**; to change a decision, add a
new ADR that supersedes it rather than rewriting history.

| ADR | Title | Status |
|-----|-------|--------|
| [0001](0001-uuid-universal-key.md) | The mtgmatcher UUID is the universal key | Accepted |
| [0002](0002-global-immutable-matcher-backend.md) | Global, immutable-after-load, unsynchronized matcher backend | Superseded by ADR-0003 |
| [0003](0003-atomic-backend-snapshots.md) | Atomically publish immutable backend snapshots | Accepted |

## Format

`Status / Context / Decision / Alternatives / Consequences / Action items`.
Statuses: Proposed · Accepted · Deprecated · Superseded by ADR-NNNN.
