#!/bin/bash
# dump.sh CHECKOUT OUT: every fixture Backend and every published datastore's
# hash for one checkout. P=dir swaps where <game>.json is read from.
set -euo pipefail
here=$(cd "$(dirname "$0")" && pwd)
checkout=$1 out=$2
cd "$checkout"
trap 'rm -f */zz_dump_test.go mtgmatcher/*/zz_dump_test.go' EXIT
python3 "$here/gen_dump.py" "$checkout" >/dev/null
rm -rf "$out" && mkdir -p "$out"
set -a; . "${ENV_FILE:-$HOME/src/go-mtgban/.env}"; set +a
if [ -n "${P:-}" ]; then
  for g in lorcana riftbound onepiece yugioh fleshandblood pokemon gundam palworld; do
    export "$(echo $g | tr a-z A-Z)_PATH=$P/$g.json"
  done
fi
pkgs=$(ls -d */zz_dump_test.go mtgmatcher/*/zz_dump_test.go 2>/dev/null | xargs -n1 dirname | sed 's|^|./|')
DUMP_DIR=$out go test -count=1 -run '^TestZZDump$' $pkgs >/dev/null
echo "$(ls "$out" | wc -l | tr -d ' ') dumps in $out"
