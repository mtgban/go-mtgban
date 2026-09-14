package main

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// workflowsDir holds one bantool-<target>.yml per scheduled target, beside
// the reusable run-bantool.yml they all call.
const workflowsDir = "../../.github/workflows"

// scheduledTarget reads the target and game a workflow hands run-bantool.yml.
// They are two inputs of one `with:` block and are written together, which is
// what lets them be read without a YAML parser; a file that separates them
// fails the read rather than matching some other job's game.
var scheduledTarget = regexp.MustCompile(
	`(?m)^[ \t]+target:[ \t]*"?([a-z0-9_]+)"?[ \t]*\n[ \t]+game:[ \t]*"?([a-z]+)"?[ \t]*$`)

// TestEveryTargetIsScheduledByItsOwnWorkflow pins the registry against the
// workflows that run it, in both directions and on the game each names.
//
// The three are one contract split across two trees. bantool derives the
// datastore it loads from the target's own name, through scraperGame, while
// run-bantool.yml spends its separate `game` input on where the results go:
// the b2://mtgban-dumps/<game>/<target> path, the TCGplayer catalog beside
// that game's datastore, and the <game>.mtgban.com host told to reload. A
// file whose two inputs disagree therefore prices one game and publishes it
// under another's name, with nothing failing anywhere - the run is green, the
// dump is in the wrong place, and the site that asked for it reloads nothing.
//
// The other two directions are cheaper to spot but not free: an entry no
// workflow names is a flag that only ever runs by hand, and a workflow naming
// no entry fails at `bantool -<target>` with a flag error, twice a day.
func TestEveryTargetIsScheduledByItsOwnWorkflow(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join(workflowsDir, "bantool-*.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatalf("no bantool workflow found under %s", workflowsDir)
	}

	scheduled := map[string]string{}
	for _, path := range paths {
		file := filepath.Base(path)
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		matches := scheduledTarget.FindAllStringSubmatch(string(body), -1)
		if len(matches) != 1 {
			t.Errorf("%s states a target with its game %d times, want once", file, len(matches))
			continue
		}
		target, game := matches[0][1], matches[0][2]

		if want := "bantool-" + target + ".yml"; file != want {
			t.Errorf("%s schedules %s, which belongs in %s", file, target, want)
		}
		if options[target] == nil {
			t.Errorf("%s schedules %s, which is not a registered target", file, target)
			continue
		}
		if got := scraperGame(target); got != game {
			t.Errorf("%s runs %s, whose name says %s, under game %q: the prices would "+
				"be published under the wrong game", file, target, got, game)
		}
		if other, seen := scheduled[target]; seen {
			t.Errorf("%s and an earlier workflow both schedule %s (%s)", file, target, other)
		}
		scheduled[target] = file
	}

	for target := range options {
		if scheduled[target] == "" {
			t.Errorf("%s is registered but no workflow schedules it", target)
		}
	}
}
