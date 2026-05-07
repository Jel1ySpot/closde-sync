package runtime

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func ResolvePreloadFile(dataDir string, appVersion string) (string, error) {
	if explicit := strings.TrimSpace(os.Getenv("CLOSDE_PRELOAD_FILE")); explicit != "" {
		if FileExists(explicit) {
			return explicit, nil
		}
		return "", fmt.Errorf("CLOSDE_PRELOAD_FILE does not exist: %s", explicit)
	}

	fileVersion, err := normalizePreloadFileVersion(appVersion)
	if err != nil {
		return "", err
	}
	target := filepath.Join(dataDir, fmt.Sprintf("preload_v%s.js", fileVersion))

	if FileExists(target) {
		return target, nil
	}

	if err := DownloadPreloadFile(target, appVersion); err != nil {
		return "", err
	}

	return target, nil
}

func DownloadPreloadFile(targetPath string, appVersion string) error {
	version, err := normalizeReleaseTag(appVersion)
	if err != nil {
		return err
	}

	downloadURL := fmt.Sprintf("https://github.com/Jel1ySpot/closde-sync/releases/download/%s/preload.js", version)
	client := &http.Client{Timeout: 30 * time.Second}
	response, err := client.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("download preload.js from %s: %w", downloadURL, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("download preload.js from %s: unexpected status %s", downloadURL, response.Status)
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("create preload.js directory: %w", err)
	}

	output, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("create preload.js file: %w", err)
	}
	defer output.Close()

	if _, err := io.Copy(output, response.Body); err != nil {
		return fmt.Errorf("write preload.js file: %w", err)
	}

	return nil
}

func normalizeReleaseTag(appVersion string) (string, error) {
	version := strings.TrimSpace(appVersion)
	if version == "" {
		return "", fmt.Errorf("current CLI version is empty")
	}
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	return version, nil
}

func normalizePreloadFileVersion(appVersion string) (string, error) {
	version := strings.TrimSpace(appVersion)
	if version == "" {
		return "", fmt.Errorf("current CLI version is empty")
	}
	return strings.TrimPrefix(version, "v"), nil
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
