package constants

// regression_test.go pins the 6 pagination constants of pkg/constants against
// accidental drift. Adapted from internal/utils/operlog/regression_test.go.
// V130R-09 D-03-10: grew from 3 to 6 when internal/constants/pagination.go
// (MinPageSize / MaxListPageSize / MaxOptionsPageSize) merged into this package.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// expectedPaginationValues pins the documented (name -> value) mapping.
var expectedPaginationValues = map[string]int{
	"DefaultCurrent":     1,
	"DefaultPageSize":    10,
	"MaxPageSize":        200,
	"MinPageSize":        10,
	"MaxListPageSize":    100,
	"MaxOptionsPageSize": 10000,
}

// readPaginationConsts parses pagination.go and returns the map of constant
// name to value for untyped integer constants in the file.
func readPaginationConsts(filename string) (map[string]int, error) {
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

// TestPaginationConstantStability asserts each pagination constant is pinned
// to its documented int value. This catches accidental renumbering.
func TestPaginationConstantStability(t *testing.T) {
	t.Parallel()
	actual, err := readPaginationConsts("pagination.go")
	if err != nil {
		t.Fatalf("failed to parse pagination.go: %v", err)
	}
	if len(actual) == 0 {
		t.Fatal("no constants found — pagination.go parser is broken")
	}
	for name, want := range expectedPaginationValues {
		got, ok := actual[name]
		if !ok {
			t.Errorf("constant %q is missing from pagination.go", name)
			continue
		}
		if got != want {
			t.Errorf("constant %q = %d, want %d", name, got, want)
		}
	}
	for name, got := range actual {
		if _, ok := expectedPaginationValues[name]; !ok {
			t.Errorf("unexpected constant %q = %d", name, got)
		}
	}
}

// TestPaginationConstantCount asserts exactly 6 constants exist.
func TestPaginationConstantCount(t *testing.T) {
	t.Parallel()
	const want = 6
	actual, err := readPaginationConsts("pagination.go")
	if err != nil {
		t.Fatalf("failed to parse pagination.go: %v", err)
	}
	if got := len(actual); got != want {
		t.Errorf("pagination constant count = %d, want %d", got, want)
	}
}
