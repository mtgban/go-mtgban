# cardmarket-market self-hosted runner

A GitHub Actions self-hosted runner, labeled `cardmarket-market`, for the
three `cardmarket_market` workflows too large for `ubuntu-latest`'s ~5-6h
practical job ceiling even filtered: Magic, Pokemon and YuGiOh (see each
workflow's own comment). Deployed as a DigitalOcean Droplet
(`s5-2vcpu-6gb-30gb`, 2 vCPU, 6GB RAM, 30GB disk, $44.64/mo), provisioned
once from `provision.sh` as cloud-init user-data - no Dockerfile, no
managed platform, a plain persistent VM running the runner as a systemd
service.

## Why a Droplet, not App Platform

Started on App Platform (`.github/runner`'s earlier history), sized
`basic-xs` then `basic-s` - both measured wrong. Magic's first real run
OOM'd during `go install` alone on `basic-xs` (1GB); doubling to `basic-s`
(2GB) got past the build but died mid-catalog-load instead, still
climbing when it did (sampled `/v2/monitoring/metrics/apps/memory_percentage`
through both crashes - see the PRs that made each change for the full
numbers). Moved to a Droplet for two reasons neither App Platform tier
fixed:

- **`deploy_on_push` killed two real runs outright.** App Platform
  redeploys on every push to `master`, anywhere in the monorepo,
  regardless of `source_dir` - a known, still-open DO limitation, not
  something the app spec could work around short of disabling it
  entirely. A Droplet has no equivalent mechanism at all: it only
  changes when something explicitly touches it. Not a workaround - the
  whole failure class doesn't exist here.
- **Debugging the OOMs meant reflecting into DO's Monitoring API** with
  ~2-minute sampling and a real blind spot right around each crash. On a
  Droplet, `ssh` and `free -m`/`htop` watch it live, and
  `doctl compute droplet-action resize` grows it in place if 6GB turns
  out not to be enough either - no app-spec redeploy cycle needed to
  find out.

The registration model is simpler here too: App Platform's containers
are stateless and get rebuilt on every deploy, so the old `entrypoint.sh`
re-registered fresh on every start and deregistered on every stop. A
Droplet is a persistent VM - `provision.sh` registers once at boot and
installs the runner as a systemd service (`svc.sh`, the runner's own
installer), which just restarts itself on crash or reboot and keeps
using the same on-disk registration rather than needing a fresh one each
time.

## Why a GitHub App instead of a PAT

Authenticates as a GitHub App (`mtgban-cardmarket-runner`, App id
`4953260`, installed on `mtgban/go-mtgban` alone with
`Administration: Read and write` and nothing else) rather than a
personal access token. `mtgban` enforces a maximum PAT lifetime
org-wide, so a PAT here would need a standing reminder to rotate before
it silently expired - not on the next restart necessarily (the running
service never re-touches its credential once registered), but on some
later one, possibly during a schedule nobody's watching closely. A
GitHub App's private key carries no forced expiration; it's revoked or
rotated only when someone actually means to. The key itself never calls
the GitHub API directly - `provision.sh` signs a ten-minute JWT with it
once, trades that for an hour-long installation token, and uses that to
fetch a registration token for `config.sh` - the whole dance happens
once at boot, not on every job.

## One-time setup

Already done for `mtgban-cardmarket-runner` (App id `4953260`,
installation id `161916144`, both already in `provision.sh` - ids alone
grant nothing without the private key, safe to commit). What's left:

1. **Create the Droplet**: from a local, uncommitted copy of
   `provision.sh` with `GH_APP_PRIVATE_KEY_B64`'s placeholder replaced by
   the same base64'd private key the App Platform deploy used
   (`base64 < private-key.pem | tr -d '\n'` of the App's downloaded
   `.pem` - the same App, so the same key works; no need to generate a
   new one) -

   ```
   doctl compute droplet create cardmarket-market \
     --region sfo3 \
     --size s5-2vcpu-6gb-30gb \
     --image ubuntu-24-04-x64 \
     --ssh-keys <your SSH key fingerprint(s), comma-separated> \
     --user-data-file <path to your local, filled-in copy of provision.sh>
   ```

   Never commit that local copy.
2. Confirm the runner shows up at
   <https://github.com/mtgban/go-mtgban/settings/actions/runners> - idle,
   labeled `cardmarket-market`. Cloud-init takes a minute or two to run
   before it appears.

To rotate the private key later (only ever a deliberate choice, nothing
forces it): generate a new one from the App's settings page, `ssh` in,
stop the service (`sudo ./svc.sh stop` in `/home/runner`), re-run the
registration portion of `provision.sh` by hand with the new key, then
`sudo ./svc.sh start` again.

To resize if 6GB isn't enough either: `doctl compute droplet-action
resize <droplet-id> --size <new-size> --resize-disk` (add
`--resize-disk` only if the new size's disk is also larger; the runner's
own registration survives a resize untouched, it's the same disk).

## Unattended upgrades don't kill an in-flight job

Ubuntu's `unattended-upgrades` installs security updates daily regardless
of whether a job is running - harmless on its own, dpkg replacing a file
on disk doesn't touch an already-running process. The real risk is
`needrestart`'s default policy of auto-restarting services it thinks are
using now-stale libraries: it silently killed a 6h Magic run mid-flight
this way once, bouncing the runner service (twice) right underneath an
in-flight job with no warning, logged only as `##[error]The runner has
received a shutdown signal`.

`provision.sh` now sets `needrestart` to list-only
(`$nrconf{restart} = 'l'`) so it never auto-restarts anything - packages
still update on schedule, the runner just doesn't get bounced by
needrestart itself. So the runner still eventually picks up patched
libraries (list-only alone would leave it running stale ones
indefinitely), the runner's own `ACTIONS_RUNNER_HOOK_JOB_STARTED`/
`_JOB_COMPLETED` hooks (`~runner/hooks/`, wired through `~runner/.env`,
the documented way to hand a self-hosted runner env vars a systemd unit
has no other route for) mark `/run/runner/busy` for the length of each
job, and an hourly `runner-safe-restart.timer` restarts the runner only
when that marker is absent *and* `needrestart -b` actually flags it -
never mid-job, but never stale forever either.

`/run/runner` rather than `/run` itself (root-owned, so the unprivileged
hooks can't write there directly) or somewhere under `$RUNNER_HOME`
(survives a reboot, which the marker must not): a `tmpfiles.d` entry
recreates that one directory, owned by `runner`, on every boot, so a
genuine reboot or power loss wipes the marker the same way it would
have if `ACTIONS_RUNNER_HOOK_JOB_COMPLETED` had run cleanly. A runner
process that crashes without also taking the whole Droplet down is a
separate, still-open gap - nothing here restarts a dead runner process,
so a stuck marker doesn't matter until something does.

## What's still a manual decision

Magic, Pokemon and YuGiOh's `cardmarket_market` workflows are routed to
this runner (`runs-on: '["self-hosted", "cardmarket-market"]'`) but still
`workflow_dispatch`-only - the runner removes the job-ceiling blocker, but
picking actual cron times that don't collide with the rest of the
`bantool-cardmarket` concurrency group's own schedule is a separate,
deliberate decision, not made here.
