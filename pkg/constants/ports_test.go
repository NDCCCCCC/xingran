package constants

// ports_test.go pins the SNMPPort constant of pkg/constants against accidental drift.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// expectedPortsValues pins the documented (name -> value) mapping.
var expectedPortsValues = map[string]int{
	"SNMPPort": 161,
}

// readPortsConsts parses ports.go and returns the map of constant name to value
// for untyped integer constants in the file.
func readPortsConsts(filename string) (map[string]int, error) {
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

// TestPortsConstantStability asserts each ports constant is pinned to its documented int value.
func TestPortsConstantStability(t *testing.T) {
	t.Parallel()
	actual, err := readPortsConsts("ports.go")
	if err != nil {
		t.Fatalf("failed to parse ports.go: %v", err)
	}
	if len(actual) == 0 {
		t.Fatal("no constants found — ports.go parser is broken")
	}
	for name, want := range expectedPortsValues {
		got, ok := actual[name]
		if !ok {
			t.Errorf("constant %q is missing from ports.go", name)
			continue
		}
		if got != want {
			t.Errorf("constant %q = %d, want %d", name, got, want)
		}
	}
	for name, got := range actual {
		if _, ok := expectedPortsValues[name]; !ok {
			t.Errorf("unexpected constant %q = %d", name, got)
		}
	}
}

// TestPortsConstantCount asserts exactly 1 constant exists.
func TestPortsConstantCount(t *testing.T) {
	t.Parallel()
	const want = 1
	actual, err := readPortsConsts("ports.go")
	if err != nil {
		t.Fatalf("failed to parse ports.go: %v", err)
	}
	if got := len(actual); got != want {
		t.Errorf("ports constant count = %d, want %d", got, want)
	}
}
