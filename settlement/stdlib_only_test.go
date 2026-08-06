package settlement

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestPackageImportsOnlyStdlib is the load-bearing guard on this package's
// reason for living here.
//
// SessionLease is the control-plane <-> edge wire contract: trafficcontrol
// mints leases, the multicast edge verifies them on the CMSD delivery path.
// It lives in multicast-api because that module is the one both sides already
// depend on -- but multicast's edge also builds for js/wasm, wasip1/wasm, and
// TinyGo, and the ROOT api package of this module cannot go there: locker.go
// imports xsync, util.go imports linkdata/deadlock, and types_persist.go /
// delivery_persist.go import lib/pq, which TinyGo cannot compile (the same
// constraint delivery_wasm.go was written for).
//
// Go links per PACKAGE, not per module, so this package is reachable from a
// TinyGo build precisely as long as its own imports stay stdlib. That is not a
// property anyone can see by reading go.mod -- the module requirement lists pq
// and friends either way. One convenience import here (a uuid helper, a
// logging package, xsync for the limiter) silently un-builds the edge, and the
// break would surface as a wasm build failure in a DIFFERENT repository, long
// after the commit that caused it.
//
// So: assert it here, hermetically, where the mistake would be made. This test
// needs no toolchain, no network, and no CI -- which matters, because this
// module currently has no CI workflows at all.
//
// If you are here because this test failed: the fix is almost never to add the
// import. Either keep the dependency in the caller, or move the code that
// needs it into a package the edge does not import.
func TestPackageImportsOnlyStdlib(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}

	fset := token.NewFileSet()
	production := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || filepath.Ext(name) != ".go" {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		if !strings.HasSuffix(name, "_test.go") {
			production++
		}
		for _, spec := range file.Imports {
			path, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				t.Fatalf("%s: bad import path %s: %v", name, spec.Path.Value, err)
			}
			// Standard-library import paths never have a dot in their first
			// segment; every module path does, because it starts with a host.
			first, _, _ := strings.Cut(path, "/")
			if strings.Contains(first, ".") {
				t.Errorf("%s imports %q: this package must stay stdlib-only so it "+
					"remains buildable for js/wasm, wasip1/wasm, and TinyGo on the "+
					"multicast edge. The rest of this module (xsync, linkdata/deadlock, "+
					"lib/pq) cannot be linked there.", name, path)
			}
		}
	}

	// Guard the guard. Counting every .go file would make this inert: this test
	// file is itself a .go file in this directory, so the count can never reach
	// zero while the test is running. Only NON-test files are counted, so the
	// case that actually matters -- the production source being renamed, deleted,
	// or excluded by a build tag, leaving the loop above with nothing real to
	// inspect -- still fails loudly instead of passing vacuously.
	if production == 0 {
		t.Fatal("no non-test .go files in this package: the import scan above had no " +
			"production source to check, so its silence means nothing")
	}
}
