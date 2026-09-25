#!/usr/bin/env python3
# gen_dump.py ROOT: write zz_dump_test.go into every package holding
# {"data": ...} fixtures. The test loads each fixture through mtgmatcher.Open
# and writes its Backend as JSON to $DUMP_DIR, and writes the sha256 of each
# published datastore's Backend beside them.
import glob
import os
import re
import sys

root = sys.argv[1]
GAMES = ["lorcana", "riftbound", "onepiece", "yugioh", "fleshandblood",
         "pokemon", "gundam", "palworld"]

# Fixtures outside mtgmatcher/<game> name their game where a test opens them.
# cardmarket's datastoreBackend(t, "<game>", fixture) calls are read below;
# list the rest by hand.
BY_HAND = {
    "gamenerdz": {"snorlaxDatastore": "pokemon",
                  "respellingDatastore": "pokemon",
                  "untitledVariantDatastore": "onepiece"},
}

# Variants a test derives with strings.Replace, by hand:
# (package, label, game, Go expression).
DERIVED = [
    ("mtgmatcher/riftbound", "sealed_absent", "riftbound",
     'strings.Replace(sealedFixture, `"sealed"`, `"ignored"`, 1)'),
]

# One mask per field a change should move alone: the env var that turns it
# on, and the statement clearing it on a CardObject (co) and a Card (c).
MASKS = {
    "ZZ_MASK_OVERSIZED": ("co.IsOversized = false", "c.IsOversized = false"),
    "ZZ_MASK_LANGUAGE": ('co.Language = ""', 'c.Language = ""'),
    "ZZ_MASK_WATERMARK": ('co.Watermark = ""', 'c.Watermark = ""'),
    "ZZ_MASK_IMAGES": ("co.Images = nil", "c.Images = nil"),
}


def fixtures(pkg):
    names = []
    for path in sorted(glob.glob(f"{root}/{pkg}/*_test.go")):
        if path.endswith("zz_dump_test.go"):
            continue
        src = open(path).read()
        names += re.findall(r'^(?:const|var)\s+(\w+)\s*=\s*`\{"data"', src, re.M)
    return names


def games_of(pkg):
    found = dict(BY_HAND.get(pkg, {}))
    for path in glob.glob(f"{root}/{pkg}/*_test.go"):
        for game, name in re.findall(r'datastoreBackend\(t, "([a-z]+)", (\w+)\)', open(path).read()):
            found[name] = game
    return found


pkgs = [f"mtgmatcher/{g}" for g in GAMES]
pkgs += sorted({os.path.dirname(p)[len(root) + 1:] for p in glob.glob(f"{root}/*/*_test.go")
                if re.search(r'`\{"data"', open(p).read())} - set(pkgs))

for pkg in pkgs:
    game = pkg.split("/")[1] if pkg.startswith("mtgmatcher/") else None
    owner = games_of(pkg)
    entries = [(n, game or owner.get(n), n) for n in fixtures(pkg) if game or owner.get(n)]
    entries += [(label, g, expr) for p, label, g, expr in DERIVED if p == pkg]
    if not entries and not game:
        continue
    name = pkg.split("/")[-1]
    masks = "".join(
        f'\tif os.Getenv("{env}") != "" {{\n'
        f"\t\tfor _, co := range b.UUIDs {{\n\t\t\t{on_co}\n\t\t}}\n"
        f"\t\tfor _, set := range b.Sets {{\n\t\t\tfor i := range set.Cards {{\n"
        f"\t\t\t\tc := &set.Cards[i]\n\t\t\t\t{on_card}\n\t\t\t}}\n\t\t}}\n\t}}\n"
        for env, (on_co, on_card) in MASKS.items())
    docs = "".join(f'\t\t{{"{n}", "{g}", {e}}},\n' for n, g, e in entries)
    real = ""
    if game:
        real = (f'\tif path := os.Getenv("{game.upper()}_PATH"); path != "" {{\n'
                f'\t\tb, err := datastore.Read("{game}", path)\n'
                "\t\tif err != nil {\n\t\t\tt.Fatal(err)\n\t\t}\n"
                '\t\tsum := fmt.Sprintf("%x\\n", sha256.Sum256(zzEncode(b)))\n'
                f'\t\tif err := os.WriteFile(filepath.Join(dir, "{game}.REAL.sha256"), []byte(sum), 0o600); err != nil {{\n'
                "\t\t\tt.Fatal(err)\n\t\t}\n\t}\n")
    src = f'''package {name}

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// zzEncode drops what JSON cannot or need not carry and sorts what arrives
// in map order, so two loads of one file encode alike.
func zzEncode(b *mtgmatcher.Backend) []byte {{
	b.Logger, b.TokenPairIDByUUIDs, b.TokenPairIDByBothNames = nil, nil, nil
{masks}	b.AllSets = append([]string(nil), b.AllSets...)
	sort.Strings(b.AllSets)
	out, err := json.MarshalIndent(b, "", " ")
	if err != nil {{
		panic(err)
	}}
	return out
}}

func TestZZDump(t *testing.T) {{
	dir := os.Getenv("DUMP_DIR")
	if dir == "" {{
		t.Skip()
	}}
	for _, d := range []struct{{ name, game, doc string }}{{
{docs}	}} {{
		b, err := mtgmatcher.Open(d.game, strings.NewReader(d.doc))
		if err != nil {{
			t.Fatalf("%s: %v", d.name, err)
		}}
		if err := os.WriteFile(filepath.Join(dir, "{name}."+d.name+".json"), zzEncode(b), 0o600); err != nil {{
			t.Fatal(err)
		}}
	}}
{real}	_, _, _ = sha256.Sum256, fmt.Sprint, datastore.Read
}}
'''
    open(f"{root}/{pkg}/zz_dump_test.go", "w").write(src)
    print(f"{pkg}: {len(entries)} fixtures" + (" + published" if game else ""))
