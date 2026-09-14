package main

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// storeFromTarget mirrors the suffix-stripping run-bantool.yml's own shell
// does before passing -target to bantool: the store's own name is all a
// workflow's target ever was, before it was told apart from another
// game's copy of the same store by a "_<game>" suffix. It lives only
// here, to check that arithmetic against every scheduled workflow's real
// (target, game) pair - bantool's own -target takes a store's name
// literally, with no decoding of its own, now that -game already says
// the game.
func storeFromTarget(target, game string) string {
	if game == "magic" {
		return target
	}
	return strings.TrimSuffix(target, "_"+game)
}

var withInputRe = regexp.MustCompile(`(?m)^\s*(target|game):\s*"?([\w-]+)"?\s*$`)

// TestScheduledWorkflowsResolveToARegisteredStore checks run-bantool.yml's
// shell suffix-stripping against every (target, game) pair a scheduled
// workflow actually passes, not just a handful of examples: every
// bantool-*.yml calls run-bantool.yml with both already stated, and the
// shell's whole job is reversing what one of them says without getting
// it wrong for a store whose own name already has an underscore in it
// (tcg_index, manapool_sealed, and the like).
func TestScheduledWorkflowsResolveToARegisteredStore(t *testing.T) {
	matches, err := filepath.Glob("../../.github/workflows/bantool-*.yml")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) == 0 {
		t.Fatal("no bantool-*.yml workflow files found")
	}

	for _, path := range matches {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}

		found := withInputRe.FindAllStringSubmatch(string(data), -1)
		var target, game string
		for _, m := range found {
			switch m[1] {
			case "target":
				target = m[2]
			case "game":
				game = m[2]
			}
		}
		if target == "" || game == "" {
			t.Errorf("%s: could not find both target and game inputs", filepath.Base(path))
			continue
		}

		store := storeFromTarget(target, game)
		opt, ok := options[store]
		if !ok {
			t.Errorf("%s: target %q game %q resolved to store %q, not in options",
				filepath.Base(path), target, game, store)
			continue
		}
		if !slices.Contains(opt.Supports, game) {
			t.Errorf("%s: target %q resolved to store %q, which does not support game %q (supports %v)",
				filepath.Base(path), target, store, game, opt.Supports)
		}
	}
}
