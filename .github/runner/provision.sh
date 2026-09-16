#!/bin/bash
# cloud-init user-data for the cardmarket-market self-hosted runner Droplet.
# Registers once at boot and installs the runner as a systemd service
# (svc.sh, the runner's own installer) rather than the register-on-every-
# start dance the old App Platform container needed - a Droplet is a
# persistent VM, not a container an unrelated push could silently rebuild
# out from under a multi-hour job, so there is no reason to re-register on
# every start here. The service restarts itself on crash or reboot,
# picking the same registration back up from disk.
#
# GH_APP_PRIVATE_KEY_B64 below is a placeholder - DO NOT commit a real key
# here. Fill in a local, uncommitted copy of this file
# (base64 < private-key.pem | tr -d '\n') before passing it as
# --user-data-file to doctl compute droplet create.
set -euo pipefail

GH_REPO="mtgban/go-mtgban"
RUNNER_LABELS="cardmarket-market"
GH_APP_ID="4953260"
GH_APP_INSTALLATION_ID="161916144"
GH_APP_PRIVATE_KEY_B64="REPLACE_ME"
RUNNER_VERSION="2.337.0"

apt-get update
apt-get install -y --no-install-recommends curl ca-certificates git jq tar openssl

id -u runner &>/dev/null || useradd -m -s /bin/bash runner

RUNNER_HOME=/home/runner
cd "$RUNNER_HOME"
curl -fsSL -o actions-runner.tar.gz \
    "https://github.com/actions/runner/releases/download/v${RUNNER_VERSION}/actions-runner-linux-x64-${RUNNER_VERSION}.tar.gz"
tar xzf actions-runner.tar.gz
rm actions-runner.tar.gz
./bin/installdependencies.sh
chown -R runner:runner "$RUNNER_HOME"

KEY_PATH="$(mktemp)"
printf '%s' "${GH_APP_PRIVATE_KEY_B64}" | base64 -d > "${KEY_PATH}"
chmod 600 "${KEY_PATH}"

b64url() { openssl base64 -e -A | tr '+/' '-_' | tr -d '='; }

# app_jwt mints a fresh App-level JWT, valid ten minutes - GitHub's own cap.
app_jwt() {
    local now iat exp header payload signing_input
    now=$(date +%s)
    iat=$((now - 60))
    exp=$((now + 600))
    header=$(printf '{"alg":"RS256","typ":"JWT"}' | b64url)
    payload=$(printf '{"iat":%d,"exp":%d,"iss":"%s"}' "${iat}" "${exp}" "${GH_APP_ID}" | b64url)
    signing_input="${header}.${payload}"
    printf '%s.%s' "${signing_input}" \
        "$(printf '%s' "${signing_input}" | openssl dgst -sha256 -sign "${KEY_PATH}" -binary | b64url)"
}

install_token() {
    curl -sf -X POST \
        -H "Authorization: Bearer $(app_jwt)" \
        -H "Accept: application/vnd.github+json" \
        "https://api.github.com/app/installations/${GH_APP_INSTALLATION_ID}/access_tokens" \
        | jq -er .token
}

REG_TOKEN=$(curl -sf -X POST \
    -H "Authorization: Bearer $(install_token)" \
    -H "Accept: application/vnd.github+json" \
    "https://api.github.com/repos/${GH_REPO}/actions/runners/registration-token" \
    | jq -er .token)
rm -f "${KEY_PATH}"

export GH_REPO RUNNER_LABELS REG_TOKEN
sudo -u runner --preserve-env=GH_REPO,RUNNER_LABELS,REG_TOKEN -H bash -c '
cd "$HOME" && ./config.sh \
    --url "https://github.com/$GH_REPO" \
    --token "$REG_TOKEN" \
    --name cardmarket-market-droplet \
    --labels "$RUNNER_LABELS" \
    --unattended \
    --replace'

cd "$RUNNER_HOME"

# Ubuntu's unattended-upgrades installs security updates daily regardless
# of whether a job is running - fine on its own, dpkg replacing files on
# disk does not touch an already-running process. The problem is
# needrestart's default policy of auto-restarting services it thinks are
# using now-stale libraries: it silently killed a 6h Magic run mid-flight
# this way once, restarting the runner service (twice) right underneath
# an in-flight job with no warning. list-only stops that outright -
# packages still update on schedule, the runner just never gets bounced
# by needrestart itself.
sed -i "s/^#\$nrconf{restart} = 'i';/\$nrconf{restart} = 'l';/" /etc/needrestart/needrestart.conf

# So the runner still picks up patched libraries eventually (list-only
# alone would leave it running stale ones indefinitely), job hooks mark
# a busy file for the length of each job - the only thing runner-safe-
# restart.timer (below) needs to know to restart safely only between
# jobs, never during one.
mkdir -p "$RUNNER_HOME/hooks"
cat > "$RUNNER_HOME/hooks/job-started.sh" << 'HOOKEOF'
#!/bin/bash
# Marks the runner busy for runner-safe-restart.sh, via ACTIONS_RUNNER_HOOK_JOB_STARTED.
touch /run/runner-busy
HOOKEOF
cat > "$RUNNER_HOME/hooks/job-completed.sh" << 'HOOKEOF'
#!/bin/bash
# Clears the busy marker, via ACTIONS_RUNNER_HOOK_JOB_COMPLETED.
rm -f /run/runner-busy
HOOKEOF
chmod +x "$RUNNER_HOME/hooks/job-started.sh" "$RUNNER_HOME/hooks/job-completed.sh"
chown -R runner:runner "$RUNNER_HOME/hooks"

# config.sh above already wrote a .env with LANG=C.UTF-8 - the runner
# service (runsvc.sh -> RunnerService.js) loads this file into its own
# environment at every start, which is the documented way to hand a
# self-hosted runner env vars a systemd unit has no other route for.
{
    echo "ACTIONS_RUNNER_HOOK_JOB_STARTED=$RUNNER_HOME/hooks/job-started.sh"
    echo "ACTIONS_RUNNER_HOOK_JOB_COMPLETED=$RUNNER_HOME/hooks/job-completed.sh"
} >> "$RUNNER_HOME/.env"

cat > /usr/local/sbin/runner-safe-restart.sh << 'SCRIPTEOF'
#!/bin/bash
# Restarts the GitHub Actions runner service, but only when it is not
# mid-job (/run/runner-busy) and needrestart actually flags it as
# running against stale, upgraded libraries.
set -euo pipefail
[ -e /run/runner-busy ] && exit 0
needrestart -b 2>/dev/null | grep -q '^NEEDRESTART-SVC: actions\.runner\.' || exit 0
systemctl restart 'actions.runner.*'
SCRIPTEOF
chmod +x /usr/local/sbin/runner-safe-restart.sh

cat > /etc/systemd/system/runner-safe-restart.service << 'UNITEOF'
[Unit]
Description=Restart the GitHub Actions runner if needrestart flags it, only when idle

[Service]
Type=oneshot
ExecStart=/usr/local/sbin/runner-safe-restart.sh
UNITEOF
cat > /etc/systemd/system/runner-safe-restart.timer << 'TIMEREOF'
[Unit]
Description=Periodic check for runner-safe-restart.service

[Timer]
OnCalendar=hourly
Persistent=true

[Install]
WantedBy=timers.target
TIMEREOF
systemctl daemon-reload
systemctl enable --now runner-safe-restart.timer

./svc.sh install runner
./svc.sh start
