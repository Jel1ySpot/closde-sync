package runtime

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ResolvePreloadFile(dataDir string) (string, error) {
	if explicit := strings.TrimSpace(os.Getenv("CLOSDE_PRELOAD_FILE")); explicit != "" {
		if FileExists(explicit) {
			return explicit, nil
		}
		return "", fmt.Errorf("CLOSDE_PRELOAD_FILE does not exist: %s", explicit)
	}

	target := filepath.Join(dataDir, "preload.js")

	if FileExists(target) {
		return target, nil
	}

	return "", errors.New("unable to locate dist/preload.js; build preload first or set CLOSDE_PRELOAD_FILE")
}

func dedupePaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		cleaned := filepath.Clean(path)
		if _, ok := seen[cleaned]; ok {
			continue
		}
		seen[cleaned] = struct{}{}
		result = append(result, cleaned)
	}
	return result
}
