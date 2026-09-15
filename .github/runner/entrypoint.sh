#!/usr/bin/env bash
# Registers a persistent runner against GH_REPO on every container start
# and deregisters it on shutdown, rather than baking in a registration
# token: the token this needs (a short-lived registration token) expires
# in about an hour, so it can only ever be fetched fresh at start. This
# also means a stateless container - a redeploy, an OOM, a platform
# restart - never leaves a stale, half-registered runner behind; it just
# registers again next boot. --replace additionally covers the case where
# the previous container's own deregistration on the way out didn't get
# to run.
#
# Authenticates as a GitHub App rather than a personal access token: an
# org can force a PAT to expire on a clock neither of us controls, and
# ties the credential to whoever happened to create it. An App's own
# private key has no forced expiration - only revoked or rotated
# deliberately. The key never touches the GitHub API directly; it signs a
# short-lived JWT (ten minutes, GitHub's own cap) that is exchanged for an
# installation access token (about an hour), which is what actually
# authenticates the calls below. That exchange happens fresh right before
# each use rather than once at the top: a Market job can run up to 12h, so
# the token minted for registration at start would be long expired by the
# time shutdown needs one of its own for deregistration.
set -euo pipefail

: "${GH_REPO:?GH_REPO must be set, e.g. mtgban/go-mtgban}"
: "${GH_APP_ID:?GH_APP_ID must be set}"
: "${GH_APP_INSTALLATION_ID:?GH_APP_INSTALLATION_ID must be set}"
: "${GH_APP_PRIVATE_KEY_B64:?GH_APP_PRIVATE_KEY_B64 must be set - base64 of the App private key, never logged or persisted past this script}"
RUNNER_LABELS="${RUNNER_LABELS:-cardmarket-market}"
RUNNER_NAME="${RUNNER_NAME:-cardmarket-market-$(hostname)}"

KEY_PATH="$(mktemp)"
printf '%s' "${GH_APP_PRIVATE_KEY_B64}" | base64 -d > "${KEY_PATH}"
chmod 600 "${KEY_PATH}"

b64url() { openssl base64 -e -A | tr '+/' '-_' | tr -d '='; }

# app_jwt mints a fresh App-level JWT, valid ten minutes - short enough
# that one leaked in a log would already be useless.
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

# install_token exchanges a fresh app_jwt for an installation access
# token - called right before each use below, never cached, since the
# hour it is valid for is far shorter than a run can take.
install_token() {
    curl -sf -X POST \
        -H "Authorization: Bearer $(app_jwt)" \
        -H "Accept: application/vnd.github+json" \
        "https://api.github.com/app/installations/${GH_APP_INSTALLATION_ID}/access_tokens" \
        | jq -er .token
}

api() {
    curl -sf -X POST \
        -H "Authorization: Bearer $(install_token)" \
        -H "Accept: application/vnd.github+json" \
        "https://api.github.com/repos/${GH_REPO}/actions/runners/$1"
}

REG_TOKEN=$(api registration-token | jq -er .token)

./config.sh \
    --url "https://github.com/${GH_REPO}" \
    --token "${REG_TOKEN}" \
    --name "${RUNNER_NAME}" \
    --labels "${RUNNER_LABELS}" \
    --unattended \
    --replace

cleanup() {
    echo "Deregistering ${RUNNER_NAME}..."
    REMOVE_TOKEN=$(api remove-token | jq -er .token) || { rm -f "${KEY_PATH}"; return 0; }
    ./config.sh remove --token "${REMOVE_TOKEN}" || true
    rm -f "${KEY_PATH}"
}
trap cleanup EXIT INT TERM

./run.sh
