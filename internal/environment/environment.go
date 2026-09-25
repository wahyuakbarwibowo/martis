// Package environment loads dotenv files without executing shell code.
package environment

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var variable = regexp.MustCompile(`\{\{\s*([A-Za-z_][A-Za-z0-9_]*)\s*\}\}`)

func Load(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	values := map[string]string{}
	scanner := bufio.NewScanner(f)
	for line := 1; scanner.Scan(); line++ {
		s := strings.TrimSpace(scanner.Text())
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}
		s = strings.TrimPrefix(s, "export ")
		key, value, ok := strings.Cut(s, "=")
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if !ok || !regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`).MatchString(key) {
			return nil, fmt.Errorf("%s:%d: invalid environment assignment", filepath.Base(path), line)
		}
		if len(value) >= 2 && (value[0] == '\'' || value[0] == '"') {
			if value[len(value)-1] != value[0] {
				return nil, fmt.Errorf("%s:%d: unclosed quote", filepath.Base(path), line)
			}
			value = value[1 : len(value)-1]
		}
		values[key] = value
	}
	return values, scanner.Err()
}

func Expand(s string, values map[string]string) (string, error) {
	var missing string
	result := variable.ReplaceAllStringFunc(s, func(match string) string {
		key := variable.FindStringSubmatch(match)[1]
		value, ok := values[key]
		if !ok {
			missing = key
			return match
		}
		return value
	})
	if missing != "" {
		return "", fmt.Errorf("undefined environment variable: %s", missing)
	}
	return result, nil
}

func Files(dir string) ([]string, error) { return filepath.Glob(filepath.Join(dir, "*.env")) }
