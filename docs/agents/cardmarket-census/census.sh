#!/bin/bash
# census.sh GAME [LABEL]: fetch one game's Cardmarket census inputs, once,
# and walk its catalog on this checkout the way Index.walkCatalog does.
# Run from the repository root. Everything lands in $OUT/GAME
# (default ~/src/claude-scratchpad/cardmarket-census/GAME):
#   data/   datastore, id map, product list, price guide, CardTrader
#           blueprints and the bridge built from them
#   dumps/  the published TCGMarket, MKMTrend, MKMLow and CT dumps
#   walk-LABEL.tsv (+ .sha), backend.json
# Delete data/ or dumps/ to fetch them again.
set -euo pipefail
here=$(cd "$(dirname "$0")" && pwd)
game=$1 label=${2:-master}
repo=$(git rev-parse --show-toplevel)
D=${OUT:-$HOME/src/claude-scratchpad/cardmarket-census}/$game
mkdir -p "$D/data" "$D/dumps"
set -a; . "${ENV_FILE:-$HOME/src/go-mtgban/.env}"; set +a
trap 'rm -f "$repo"/cardmarket/zz_census_*_test.go' EXIT

# template NAME writes harnesses.md's block for cardmarket/NAME into place.
template() {
  python3 - "$here/harnesses.md" "$1" > "$repo/cardmarket/$1" <<'EOF'
import re, sys
text = open(sys.argv[1]).read()
m = re.search(r'### `cardmarket/' + re.escape(sys.argv[2]) + r'`.*?```go\n(.*?)```', text, re.S)
sys.stdout.write(m.group(1))
EOF
}

# fetch URL DEST unpacks an .xz download into DEST, which appears only once
# whole, so an interrupted fetch is retried on the next run.
fetch() {
  case $1 in
    b2://*) b2 file download -q "$1" download.xz >/dev/null 2>&1 || return 1 ;;
    *) curl -sSf --retry 3 -o download.xz "$1" || return 1 ;;
  esac
  xz -dc download.xz > "$2.part" || return 1
  mv "$2.part" "$2"
  rm -f download.xz
}

cd "$D/data"
if [ "$game" = magic ]; then
  # Magic prices from MTGJSON's AllPrintings and its CardmarketIdentifiers,
  # the file the Cardmarket workflows read as MKMIDS_MAGIC.
  ln -sf "$ALLPRINTINGS5_PATH" magic-datastore.json
  [ -e magic-catalog.json ] || fetch https://www.mtgjson.com/api/v5/CardmarketIdentifiers.json.xz magic-catalog.json
else
  export B2_APPLICATION_KEY_ID=$B2_APPLICATION_KEY_ID_DATASTORE B2_APPLICATION_KEY=$B2_APPLICATION_KEY_DATASTORE
  [ -e "$game-datastore.json" ] || fetch "b2://mtgban-datastore/$game/$game.json.xz" "$game-datastore.json"
  [ -e "$game-catalog.json" ] || fetch "b2://mtgban-datastore/$game/cardmarket_catalog.json.xz" "$game-catalog.json"
  [ -e tcgplayer-catalog.json ] || fetch "b2://mtgban-datastore/$game/tcgplayer-catalog.json.xz" tcgplayer-catalog.json
fi

cd "$D/dumps"
export B2_APPLICATION_KEY_ID=$B2_APPLICATION_KEY_ID_DUMPS B2_APPLICATION_KEY=$B2_APPLICATION_APP_KEY_DUMPS
suffix=_$game
[ "$game" = magic ] && suffix=
for f in "tcg_index$suffix/retail/TCGMarket" "cardmarket$suffix/retail/MKMTrend" \
         "cardmarket$suffix/retail/MKMLow" "cardtrader$suffix/retail/CT"; do
  [ -e "$(basename "$f").json" ] || fetch "b2://mtgban-dumps/$game/$f.json.xz" "$(basename "$f").json" \
    || echo "nothing published at $f"
done

cd "$repo"
if [ ! -e "$D/data/$game-bridge.json" ]; then
  template zz_census_fetch_test.go
  ZZ_GAME=$game ZZ_DIR=$D/data go test -count=1 -timeout 60m -run '^TestZZCensusFetch$' -v ./cardmarket/ \
    > "$D/fetch.log" 2>&1 || { tail -5 "$D/fetch.log"; exit 1; }
  grep -o '[0-9]* list.*' "$D/fetch.log"
fi
template zz_census_walk_test.go
ZZ_GAME=$game ZZ_DIR=$D/data ZZ_OUT=$D/walk-$label.tsv go test -count=1 -timeout 60m -run '^TestZZCensusWalk$' -v ./cardmarket/ \
  > "$D/walk-$label.log" 2>&1 || { tail -5 "$D/walk-$label.log"; exit 1; }
git rev-parse HEAD > "$D/walk-$label.sha"
if [ ! -e "$D/backend.json" ]; then
  template zz_census_backend_test.go
  ZZ_GAME=$game ZZ_DIR=$D/data ZZ_OUT=$D/backend.json go test -count=1 -run '^TestZZCensusBackend$' ./cardmarket/ \
    > "$D/backend.log" 2>&1 || { tail -5 "$D/backend.log"; exit 1; }
fi
grep -o '[0-9]* products.*' "$D/walk-$label.log"
echo "walk: $D/walk-$label.tsv at $(git rev-parse --short HEAD)"
