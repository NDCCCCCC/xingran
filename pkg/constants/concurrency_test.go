package constants

// concurrency_test.go pins the CommandConcurrency constant of pkg/constants against accidental drift.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// expectedConcurrencyValues pins the documented (name -> value) mapping.
var expectedConcurrencyValues = map[string]int{
	"CommandConcurrency": 10,
}

// readConcurrencyConsts parses concurrency.go and returns the map of constant name to value
// for untyped integer constants in the file.
func readConcurrencyConsts(filename string) (map[string]int, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, 0)
	if err != nil {
		return nil, err
	}
	result := make(map[string]int)
	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.CONST {
			continue
		}
		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range valueSpec.Names {
				if len(valueSpec.Values) <= i {
					continue
				}
				lit, ok := valueSpec.Values[i].(*ast.BasicLit)
				if !ok || lit.Kind != token.INT {
					continue
				}
				var val int
				for _, c := range lit.Value {
					val = val*10 + int(c-'0')
				}
				result[name.Name] = val
			}
		}
	}
	return result, nil
}

// TestConcurrencyConstantStability asserts each concurrency constant is pinned to its documented int value.
func TestConcurrencyConstantStability(t *testing.T) {
	t.Parallel()
	actual, err := readConcurrencyConsts("concurrency.go")
	if err != nil {
		t.Fatalf("failed to parse concurrency.go: %v", err)
	}
	if len(actual) == 0 {
		t.Fatal("no constants found — concurrency.go parser is broken")
	}
	for name, want := range expectedConcurrencyValues {
		got, ok := actual[name]
		if !ok {
			t.Errorf("constant %q is missing from concurrency.go", name)
			continue
		}
		if got != want {
			t.Errorf("constant %q = %d, want %d", name, got, want)
		}
	}
	for name, got := range actual {
		if _, ok := expectedConcurrencyValues[name]; !ok {
			t.Errorf("unexpected constant %q = %d", name, got)
		}
	}
}

// TestConcurrencyConstantCount asserts exactly 1 constant exists.
func TestConcurrencyConstantCount(t *testing.T) {
	t.Parallel()
	const want = 1
	actual, err := readConcurrencyConsts("concurrency.go")
	if err != nil {
		t.Fatalf("failed to parse concurrency.go: %v", err)
	}
	if got := len(actual); got != want {
		t.Errorf("concurrency constant count = %d, want %d", got, want)
	}
}
