package component_collector

import (
	"fmt"
	"os"
	"path/filepath"
)

// fixturesRoot returns the absolute path to the templates/samples directory.
//
// templates/ 现位于 internal/templates/embedded/templates（与 go:embed 同包约束）。
// fixturesRoot 从项目根目录出发定位该子目录，兼容从任意包目录运行 go test。
func fixturesRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getcwd: %w", err)
	}
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return filepath.Join(dir, "internal", "templates", "embedded", "templates", "samples"), nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("project root (go.mod) not found from %s", cwd)
}

// LoadFixture reads templates/samples/<name> and returns its content as a
// string. Empty result + non-nil error when the file is missing or the
// project root cannot be located.
//
// Used by the CLI collector tests to load real-machine CLI samples
// (29 huawei + 6 ruijie files at the time of writing).
func LoadFixture(name string) (string, error) {
	root, err := fixturesRoot()
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(filepath.Join(root, name))
	if err != nil {
		return "", fmt.Errorf("read fixture %s: %w", name, err)
	}
	return string(b), nil
}

// CountFixtures returns the number of *.txt files in templates/samples.
// The count is computed dynamically via filepath.Glob so that adding or
// removing samples does not require updating test code (RESEARCH §Wave 0
// WARNING 3 mitigation — do NOT hardcode the current count).
//
// Returns an error only if the samples directory cannot be located or
//_glob fails (e.g. malformed pattern — impossible with a literal).
func CountFixtures() (int, error) {
	root, err := fixturesRoot()
	if err != nil {
		return 0, err
	}
	matches, err := filepath.Glob(filepath.Join(root, "*.txt"))
	if err != nil {
		return 0, err
	}
	return len(matches), nil
}
