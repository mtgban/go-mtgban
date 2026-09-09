// Package testrules holds checks about how the suites are written rather
// than about what the code does.
package testrules

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// loaders are the calls that read a datastore off disk or out of a bucket.
// Reaching one of these costs about ten seconds and three gigabytes.
var loaders = []string{
	"datastore.Read", "datastore.Load", "datastore.Open", "simplecloud.Open",
	"magic.Load", "mtgmatcher.Open",
}

// TestMainDoesNotParseADatastore is the rule that keeps a package's tests
// costing what they use.
//
// A datastore read in TestMain runs before the first test, so it is paid by
// every run of the package - including one selecting a test that never looks
// at a card, and including `go build`-style checks that select nothing at
// all. Across the tree that was 158.7s of setup for suites whose own work is
// a fraction of it.
//
// Put the read behind a helper the tests call instead, guarded by a
// sync.Once, and have that helper skip where the datastore is not
// configured. Every package in this repository has one; copy the nearest.
func TestMainDoesNotParseADatastore(t *testing.T) {
	root := moduleRoot(t)
	fset := token.NewFileSet()

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			// A file this walk cannot read is the compiler's problem, not this
			// test's; it will say so with a better message than we can.
			return nil
		}
		for _, decl := range file.Decls {
			fn, isFunc := decl.(*ast.FuncDecl)
			if !isFunc || fn.Name.Name != "TestMain" || fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, isCall := n.(*ast.CallExpr)
				if !isCall {
					return true
				}
				name := callName(call.Fun)
				for _, loader := range loaders {
					if name == loader {
						rel, _ := filepath.Rel(root, path)
						t.Errorf("%s: TestMain calls %s, which every run of the package then waits for.\n"+
							"Move the read into a helper the tests that need it call, behind a sync.Once,\n"+
							"and skip there when the datastore is not configured.", rel, name)
					}
				}
				return true
			})
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// callName renders pkg.Fn for a qualified call and "" for anything else.
func callName(fun ast.Expr) string {
	sel, isSel := fun.(*ast.SelectorExpr)
	if !isSel {
		return ""
	}
	ident, isIdent := sel.X.(*ast.Ident)
	if !isIdent {
		return ""
	}
	return ident.Name + "." + sel.Sel.Name
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		_, err := os.Stat(filepath.Join(dir, "go.mod"))
		if err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("no go.mod above the test's directory")
		}
		dir = parent
	}
}
