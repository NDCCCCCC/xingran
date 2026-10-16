package constants

// protocol_test.go pins the HTTPProto and HTTPSProto constants of pkg/constants against accidental drift.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// expectedProtocolValues pins the documented (name -> value) mapping.
var expectedProtocolValues = map[string]string{
	"HTTPProto":  "http",
	"HTTPSProto": "https",
}

// readProtocolConsts parses protocol.go and returns the map of constant name to string value.
func readProtocolConsts(filename string) (map[string]string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, 0)
	if err != nil {
		return nil, err
	}
	result := make(map[string]string)
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
				if !ok || lit.Kind != token.STRING {
					continue
				}
				// Strip quotes from the string literal
				val := lit.Value
				if len(val) >= 2 {
					result[name.Name] = val[1 : len(val)-1]
				}
			}
		}
	}
	return result, nil
}

// TestProtocolConstantStability asserts each protocol constant is pinned to its documented string value.
func TestProtocolConstantStability(t *testing.T) {
	t.Parallel()
	actual, err := readProtocolConsts("protocol.go")
	if err != nil {
		t.Fatalf("failed to parse protocol.go: %v", err)
	}
	if len(actual) == 0 {
		t.Fatal("no constants found — protocol.go parser is broken")
	}
	for name, want := range expectedProtocolValues {
		got, ok := actual[name]
		if !ok {
			t.Errorf("constant %q is missing from protocol.go", name)
			continue
		}
		if got != want {
			t.Errorf("constant %q = %q, want %q", name, got, want)
		}
	}
	for name, got := range actual {
		if _, ok := expectedProtocolValues[name]; !ok {
			t.Errorf("unexpected constant %q = %q", name, got)
		}
	}
}

// TestProtocolConstantCount asserts exactly 2 constants exist.
func TestProtocolConstantCount(t *testing.T) {
	t.Parallel()
	const want = 2
	actual, err := readProtocolConsts("protocol.go")
	if err != nil {
		t.Fatalf("failed to parse protocol.go: %v", err)
	}
	if got := len(actual); got != want {
		t.Errorf("protocol constant count = %d, want %d", got, want)
	}
}
