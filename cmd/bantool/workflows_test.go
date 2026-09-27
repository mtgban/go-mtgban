package main

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// workflowsDir holds one bantool-<game>-<store>.yml per scheduled target,
// beside the reusable run-bantool.yml they all call.
const workflowsDir = "../../.github/workflows"

// scheduledStore reads the store and game a workflow hands run-bantool.yml.
// They are two inputs of one `with:` block and are written together, which is
// what lets them be read without a YAML parser; a file that separates them
// fails the read rather than matching some other job's game.
var scheduledStore = regexp.MustCompile(
	`(?m)^[ \t]+store:[ \t]*"?([a-z0-9_]+)"?[ \t]*\n[ \t]+game:[ \t]*"?([a-z]+)"?[ \t]*$`)

// workflowName reads a workflow's own top-level `name:` line - the one at
// column zero, not a step's or a job's own `name:` further indented in.
var workflowName = regexp.MustCompile(`(?m)^name:[ \t]*(.+?)[ \t]*$`)

// scheduledDispatch reads a workflow's repository_dispatch types, written
// directly below the trigger line the same way scheduledStore is.
var scheduledDispatch = regexp.MustCompile(
	`(?m)^[ \t]+repository_dispatch:[ \t]*\n[ \t]+types:[ \t]*\[([^\]\n]*)\][ \t]*$`)

// TestEveryTargetIsScheduledByItsOwnWorkflow pins the registry against the
// workflows that run it, in both directions and on the game each names.
//
// The three are one contract split across two trees. bantool loads the
// datastore of the game a store is registered under, while run-bantool.yml
// spends its separate `game` input on where the results go: the
// b2://mtgban-dumps/<game>/<store> path, the TCGplayer catalog beside that
// game's datastore, and the <game>.mtgban.com host told to reload. A file
// whose two inputs disagree therefore prices one game and publishes it under
// another's name, with nothing failing anywhere - the run is green, the dump
// is in the wrong place, and the site that asked for it reloads nothing.
//
// The other two directions are cheaper to spot but not free: an entry no
// workflow names is a store that only ever runs by hand, and a workflow
// naming no entry fails at `bantool -store <store>` with a flag error, twice
// a day.
//
// Dispatch types matter too: the site reruns one store via
// `<game>-<store>`, a whole game via `<game>-all`, and GitHub silently runs
// nothing for a type no workflow lists, so a wrong one fails nowhere.
func TestEveryTargetIsScheduledByItsOwnWorkflow(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join(workflowsDir, "bantool-*.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatalf("no bantool workflow found under %s", workflowsDir)
	}

	type target struct{ game, store string }

	scheduled := map[target]string{}
	for _, path := range paths {
		file := filepath.Base(path)
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}

		matches := scheduledStore.FindAllStringSubmatch(string(body), -1)
		if len(matches) != 1 {
			t.Errorf("%s states a store with its game %d times, want once", file, len(matches))
			continue
		}
		store, game := matches[0][1], matches[0][2]

		if want := "bantool-" + game + "-" + store + ".yml"; file != want {
			t.Errorf("%s schedules %s / %s, which belongs in %s", file, game, store, want)
		}

		dispatches := scheduledDispatch.FindAllStringSubmatch(string(body), -1)
		wantTypes := game + "-" + store + ", " + game + "-all"
		switch {
		case len(dispatches) != 1:
			t.Errorf("%s states repository_dispatch types %d times, want once", file, len(dispatches))
		case dispatches[0][1] != wantTypes:
			t.Errorf("%s dispatches on %q, want %q", file, dispatches[0][1], wantTypes)
		}

		if _, ok := options[mtgmatcher.Game(game)][store]; !ok {
			t.Errorf("%s schedules %s / %s, which is not a registered target", file, game, store)
			continue
		}

		nameMatch := workflowName.FindSubmatch(body)
		wantName := game + " / " + store
		switch {
		case nameMatch == nil:
			t.Errorf("%s has no top-level name: line", file)
		case string(nameMatch[1]) != wantName:
			t.Errorf("%s names itself %q, want %q", file, nameMatch[1], wantName)
		}

		key := target{game, store}
		if other, seen := scheduled[key]; seen {
			t.Errorf("%s and an earlier workflow both schedule %s / %s (%s)", file, game, store, other)
		}
		scheduled[key] = file
	}

	for game, scrapers := range options {
		for store := range scrapers {
			key := target{string(game), store}
			if scheduled[key] == "" {
				t.Errorf("%s / %s is registered but no workflow schedules it", game, store)
			}
		}
	}
}
