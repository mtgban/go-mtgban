# cardmarket-market self-hosted runner

A GitHub Actions self-hosted runner, labeled `cardmarket-market`, for the
three `cardmarket_market` workflows too large for `ubuntu-latest`'s ~5-6h
practical job ceiling even filtered: Magic, Pokemon and YuGiOh (see each
workflow's own comment). Deployed as a DigitalOcean App Platform `worker`
component - no HTTP endpoint, just a background process DO keeps running
and restarts on crash - built from `Dockerfile` in this directory via DO's
GitHub integration, not a separately-pushed image.

## Why App Platform over a Droplet

Sized at `basic-xs` (1 shared vCPU, 1GB RAM, $10/mo): the scraping itself
is light (Market walks its catalog at concurrency 1, mostly blocked on
network I/O), but each job does a fresh `go install` of bantool and loads
Magic's full catalog into `mtgmatcher` - the one tier up from the $5 floor
buys headroom against an OOM burning hours of live API quota for nothing,
cheap insurance either way.

App Platform's `worker` component is a plain long-running background
process, the same shape a GitHub Actions runner already is, so nothing
about the runner's own design needs to change to run here versus a
Droplet - the tradeoff is a Droplet's registration survives a restart on
disk, where App Platform's containers are stateless and re-register fresh
every time. `entrypoint.sh` is written around that: it fetches a new
registration token from the GitHub API at every start and deregisters on
shutdown, so a redeploy or a crash never leaves a stale runner behind for
the next start to reconcile.

## Why a GitHub App instead of a PAT

`entrypoint.sh` authenticates as a GitHub App
(`mtgban-cardmarket-runner`, App id `4953260`, installed on
`mtgban/go-mtgban` alone with `Administration: Read and write` and
nothing else) rather than a personal access token. `mtgban` enforces a
maximum PAT lifetime org-wide, so a PAT here would need a standing
reminder to rotate before it silently expired - probably not on the next
restart, since the running container never re-touches its credential
between start and stop, but on some later one, possibly during a
schedule nobody's watching closely. A GitHub App's private key carries no
forced expiration; it's revoked or rotated only when someone actually
means to. The key itself never calls the GitHub API directly - it signs
a ten-minute JWT, which entrypoint.sh trades for an hour-long
installation token right before each of the two calls that need one
(registration, then again at deregistration, since a run can last up to
12h and the first token would be long expired by the time the second is
needed).

## One-time setup

Already done for `mtgban-cardmarket-runner` (App id `4953260`,
installation id `161916144`, both already in `app.yaml` - ids alone
grant nothing without the private key, safe to commit). What's left:

1. **Deploy the app**: `doctl apps create --spec .github/runner/app.yaml`
   from a local copy of `app.yaml` with `GH_APP_PRIVATE_KEY_B64`'s
   placeholder replaced by `base64 < private-key.pem | tr -d '\n')` of
   the App's downloaded `.pem` - never commit that copy. DO's GitHub App
   is already installed org-wide on `mtgban` (confirmed via the API
   before writing this), so no separate authorization step is needed.
2. Confirm the runner shows up at
   <https://github.com/mtgban/go-mtgban/settings/actions/runners> - idle,
   labeled `cardmarket-market`.

**`deploy_on_push` is deliberately `false`.** `source_dir` scopes the
build context, not the deploy trigger - DO redeploys on every push to
`master`, anywhere in the monorepo, regardless of `source_dir` (a known,
still-open DO limitation, not something this config can work around).
Confirmed live: an unrelated workflow-file merge redeployed this app and
killed a Market job 13 minutes into a run expected to take hours, on a
repo that merges many times a day. Redeploy by hand after changing
anything in this directory:

```
doctl apps update <app-id> --spec .github/runner/app.yaml
```

using a local copy with the real `GH_APP_PRIVATE_KEY_B64` filled in, or
`doctl apps spec get <app-id>` first to round-trip the already-set secret
(it comes back as DO's own encrypted placeholder, safe to resubmit
unchanged) without ever needing the real value again.

To rotate the private key later (only ever a deliberate choice, nothing
forces it): generate a new one from the App's settings page, set the new
base64'd value the same way (`doctl apps update <app-id> --spec <local
copy>` or the dashboard's encrypted env var UI), redeploy, and revoke the
old key once the new one is confirmed working.

## What's still a manual decision

Magic, Pokemon and YuGiOh's `cardmarket_market` workflows are routed to
this runner (`runs-on: '["self-hosted", "cardmarket-market"]'`) but still
`workflow_dispatch`-only - the runner removes the job-ceiling blocker, but
picking actual cron times that don't collide with the rest of the
`bantool-cardmarket` concurrency group's own schedule is a separate,
deliberate decision, not made here.
