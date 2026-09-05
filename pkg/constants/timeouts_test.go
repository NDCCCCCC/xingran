package constants

// timeouts_test.go pins the 6 time.Duration constants of pkg/constants against
// accidental drift. Handles time.Duration expressions (X * time.Second / time.Minute).

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
	"time"
)

// expectedTimeoutsValues pins the documented (name -> Go duration string) mapping.
// Note: Go duration string uses the largest evenly-dividing unit (300s = 5m0s, 60s = 1m0s).
var expectedTimeoutsValues = map[string]string{
	"CommandExecTimeout":        "5m0s",
	"CommandReadTimeout":        "1m0s",
	"LDAPConnTimeout":           "30s",
	"ADSyncTimeout":             "30m0s",
	"SchedulerShutdownTimeout":  "5s",
	"ADSyncTaskTimeout":         "1m0s",
	"RestoreConfigTimeout":      "10m0s",
}

// readTimeoutsConsts parses timeouts.go and returns the map of constant
// name to evaluated duration string. Handles BinaryExpr (X * time.Second/Minute).
func readTimeoutsConsts(filename string) (map[string]string, error) {
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
			if !ok || len(valueSpec.Names) == 0 || len(valueSpec.Values) == 0 {
				continue
			}
			for i, name := range valueSpec.Names {
				if i >= len(valueSpec.Values) {
					continue
				}
				val := valueSpec.Values[i]
				dur, err := evalDurationExpr(val)
				if err != nil {
					continue
				}
				result[name.Name] = dur.String()
			}
		}
	}
	return result, nil
}

// evalDurationExpr evaluates an ast expr that may be a time.Duration binary expression.
// Returns the evaluated time.Duration or error.
func evalDurationExpr(expr ast.Expr) (time.Duration, error) {
	switch v := expr.(type) {
	case *ast.BinaryExpr:
		// Expect: X * time.Second or X * time.Minute
		xLit, ok := v.X.(*ast.BasicLit)
		if !ok || xLit.Kind != token.INT {
			return 0, errNotDuration
		}
		x := 0
		for _, c := range xLit.Value {
			if c < '0' || c > '9' {
				return 0, errNotDuration
			}
			x = x*10 + int(c-'0')
		}

		var unit time.Duration
		switch y := v.Y.(type) {
		case *ast.SelectorExpr:
			ident, ok := y.X.(*ast.Ident)
			if !ok {
				return 0, errNotDuration
			}
			if ident.Name != "time" {
				return 0, errNotDuration
			}
			selIdent := y.Sel
			switch selIdent.Name {
			case "Second":
				unit = time.Second
			case "Minute":
				unit = time.Minute
			case "Millisecond":
				unit = time.Millisecond
			case "Microsecond":
				unit = time.Microsecond
			case "Nanosecond":
				unit = time.Nanosecond
			default:
				return 0, errNotDuration
			}
		default:
			return 0, errNotDuration
		}
		return time.Duration(x) * unit, nil
	case *ast.Ident:
		// Fallback: try to resolve via value (should not happen for untyped const)
		return 0, errNotDuration
	}
	return 0, errNotDuration
}

var errNotDuration = &durationError{}

type durationError struct{}

func (e *durationError) Error() string { return "not a duration expression" }

// TestTimeoutsConstantStability asserts each timeout constant is pinned to its
// documented duration string value.
func TestTimeoutsConstantStability(t *testing.T) {
	t.Parallel()
	actual, err := readTimeoutsConsts("timeouts.go")
	if err != nil {
		t.Fatalf("failed to parse timeouts.go: %v", err)
	}
	if len(actual) == 0 {
		t.Fatal("no constants found — timeouts.go parser is broken")
	}
	for name, want := range expectedTimeoutsValues {
		got, ok := actual[name]
		if !ok {
			t.Errorf("constant %q is missing from timeouts.go", name)
			continue
		}
		if got != want {
			t.Errorf("constant %q = %s, want %s", name, got, want)
		}
	}
	for name, got := range actual {
		if _, ok := expectedTimeoutsValues[name]; !ok {
			t.Errorf("unexpected constant %q = %s", name, got)
		}
	}
}

// TestTimeoutsConstantCount asserts exactly 6 constants exist.
func TestTimeoutsConstantCount(t *testing.T) {
	t.Parallel()
	const want = 7
	actual, err := readTimeoutsConsts("timeouts.go")
	if err != nil {
		t.Fatalf("failed to parse timeouts.go: %v", err)
	}
	if got := len(actual); got != want {
		t.Errorf("timeouts constant count = %d, want %d", got, want)
	}
}
