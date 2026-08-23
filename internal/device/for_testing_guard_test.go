package device

import (
	"fmt"
	"io/fs"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoProductionForTestingReferences is the AST backstop of the three-layer
// test-isolation contract for `*ForTesting`-suffixed symbols (76-05 / INFRA-05):
//
//	(1) Compiler  — symbols defined in _test.go files are physically invisible
//	                to production code (separate compilation unit).
//	(2) Naming    — when a test needs to reach private fields across packages,
//	                the escape hatch is a PRODUCTION symbol explicitly suffixed
//	                `ForTesting` (see internal/device/e2e_helpers.go). The suffix
//	                is the isolation contract — there is no build tag, because
//	                A1 (54-01-PLAN.md) requires e2e tests to run on every
//	                `go test ./...`.
//	(3) AST guard — this test: any production .go file (non _test.go) that
//	                CALLS or REFERENCES a `*ForTesting`-suffixed identifier
//	                fails the build with a file:line report.
//
// Why the backstop matters: the ForTesting factories skip connection pool
// bookkeeping, device-level locking, reachability checks, and credential
// decryption. Production code calling them silently bypasses those security
// steps — the compiler cannot catch same-package misuse of e2e_helpers.go,
// so this guard does.
//
// Scan semantics (modeled on internal/models/status_constants_test.go):
//   - Walk the whole repository from "../.." (go test cwd = package dir).
//   - Skip dot-prefixed directories (.git, .claude/worktrees repo copies,
//     scripts/.archive-*), vendor, node_modules, xingran-react-frontend,
//     testdata and tests — never parse files that are not shipped source.
//     A violation whose path contains ".claude/worktrees" means the filter
//     is broken (stale repo copies would cause random reds).
//   - Parse failures FAIL the test (never silently skipped), and scanning
//     0 files also fails (guards against cwd-drift false greens).
//
// Whitelist: internal/device/e2e_helpers.go — the definition file itself is a
// production file whose internal references to its own ForTesting symbols are
// legal (function declarations are not references; only its internal call
// newScrapliWrapperForTesting(d) needs the exemption).
func TestNoProductionForTestingReferences(t *testing.T) {
	const repoRoot = "../.." // go test cwd = internal/device → repository root
	const whitelist = "internal/device/e2e_helpers.go"

	var (
		violations   []string
		scannedFiles int
	)

	walkErr := filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// The walk root itself ("../..") has Name() ".." which starts with
			// a dot but must never be skipped — only filter subdirectories.
			if path == repoRoot {
				return nil
			}
			name := d.Name()
			if strings.HasPrefix(name, ".") || // .git, .claude/worktrees copies, .archive-*
				name == "vendor" ||
				name == "node_modules" ||
				name == "xingran-react-frontend" ||
				name == "testdata" ||
				name == "tests" {
				return fs.SkipDir
			}
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}
		// Whitelist check on the repo-relative path: WalkDir paths keep the
		// "../.." prefix, so compare against filepath.Rel(repoRoot, path).
		if rel, relErr := filepath.Rel(repoRoot, path); relErr == nil &&
			filepath.ToSlash(rel) == whitelist {
			return nil
		}

		fset := token.NewFileSet()
		f, perr := parser.ParseFile(fset, path, nil, 0)
		if perr != nil {
			return fmt.Errorf("parse %s: %w", path, perr)
		}
		scannedFiles++
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.CallExpr:
				// Direct call form: poisonCallForTesting(...)
				if id, ok := x.Fun.(*ast.Ident); ok && strings.HasSuffix(id.Name, "ForTesting") {
					violations = append(violations, fmt.Sprintf(
						"%s: call to %s", fset.Position(x.Pos()), id.Name))
				}
			case *ast.SelectorExpr:
				// Qualified reference form: device.NewPooledConnectionForTesting(...)
				if strings.HasSuffix(x.Sel.Name, "ForTesting") {
					violations = append(violations, fmt.Sprintf(
						"%s: reference to %s", fset.Position(x.Pos()), x.Sel.Name))
				}
			}
			return true
		})
		return nil
	})
	if walkErr != nil {
		t.Fatalf("ForTesting guard walk failed: %v", walkErr)
	}
	if scannedFiles == 0 {
		t.Fatal("ForTesting guard scanned 0 production .go files — cwd drifted from the package directory? (false green guard, cf. status_constants_test.go)")
	}
	for _, v := range violations {
		t.Errorf("production ForTesting reference: %s", v)
	}
	if len(violations) == 0 {
		t.Logf("ForTesting guard: %d production .go files scanned, 0 violations", scannedFiles)
	}
}
