package testrules

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/internal/vocabulary"
)

// ciJob reads a job's one datastore variable and the go test line under it.
var ciJob = regexp.MustCompile(`(?m)^\s+([A-Z]+_PATH): .*\n\s+run: \|\n\s+go test (.*) -v$`)

// TestEveryGatedSuiteRunsUnderItsDatastore fails a package that reads a
// game's datastore variable when the ci.yml job exporting it does not run
// that package - otherwise the gated test just skips there forever with CI
// green. Every vocabulary.Games variable must also have matched ciJob, since
// a differently-shaped job would otherwise drop out of jobs silently and
// disable this very check for whatever it gates.
func TestEveryGatedSuiteRunsUnderItsDatastore(t *testing.T) {
	root := moduleRoot(t)
	ci, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	jobs := map[string][]string{}
	for _, m := range ciJob.FindAllStringSubmatch(string(ci), -1) {
		jobs[m[1]] = strings.Fields(m[2])
	}
	for _, env := range vocabulary.Games {
		if _, ok := jobs[env]; !ok {
			t.Errorf("no ci.yml job's go test line parsed for %s: either no job exports it, or its run block does not match ciJob's single-command shape", env)
		}
	}

	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Skip what the go tool itself skips below the root for ./... - a
		// dot- or underscore-prefixed name, or testdata - so a stray checkout
		// under e.g. .claude/worktrees never reaches this walk.
		name := d.Name()
		if path != root && (strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") || name == "testdata") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		pkg, _ := filepath.Rel(root, filepath.Dir(path))
		for env, patterns := range jobs {
			if !strings.Contains(string(body), `"`+env+`"`) {
				continue
			}
			covered := false
			for _, p := range patterns {
				dir := strings.TrimSuffix(strings.TrimPrefix(p, "./"), "/...")
				if pkg == dir || strings.HasPrefix(pkg, dir+"/") {
					covered = true
					break
				}
			}
			if !covered {
				rel, _ := filepath.Rel(root, path)
				t.Errorf("%s reads %s, but the ci.yml job exporting it does not run ./%s/...", rel, env, pkg)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
