package runtime

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const claudePackageRegistryURL = "https://registry.npmjs.org/@anthropic-ai/claude-code"

func EnsureClaudeCode(cfg Config) error {
	if FileExists(cfg.ClaudeCLIPath) {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(cfg.ClaudeRootPath), 0o755); err != nil {
		return err
	}

	archiveURL := fmt.Sprintf("https://registry.npmjs.org/@anthropic-ai/claude-code/-/claude-code-%s.tgz", cfg.ClaudeVersion)
	response, err := http.Get(archiveURL)
	if err != nil {
		return fmt.Errorf("download Claude Code %s: %w", cfg.ClaudeVersion, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("download Claude Code %s: unexpected status %s", cfg.ClaudeVersion, response.Status)
	}

	tempDir, err := os.MkdirTemp("", "closde-claude-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	if err := extractTarGz(response.Body, tempDir); err != nil {
		return err
	}

	extractedPackagePath := filepath.Join(tempDir, "package")
	if !DirExists(extractedPackagePath) {
		return fmt.Errorf("download Claude Code %s: extracted package directory missing", cfg.ClaudeVersion)
	}

	return CopyDir(extractedPackagePath, cfg.ClaudeRootPath)
}

func FetchLatestClaudeCodeVersion() (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	response, err := client.Get(claudePackageRegistryURL)
	if err != nil {
		return "", fmt.Errorf("query Claude Code registry metadata: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("query Claude Code registry metadata: unexpected status %s", response.Status)
	}

	var payload struct {
		DistTags map[string]string `json:"dist-tags"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode Claude Code registry metadata: %w", err)
	}

	latestVersion := strings.TrimSpace(payload.DistTags["latest"])
	if latestVersion == "" {
		return "", fmt.Errorf("missing latest dist-tag in Claude Code registry metadata")
	}

	return latestVersion, nil
}

func extractTarGz(reader io.Reader, targetDir string) error {
	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return err
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		cleanName := filepath.Clean(header.Name)
		if cleanName == "." || cleanName == ".." || filepath.IsAbs(cleanName) {
			continue
		}

		targetPath := filepath.Join(targetDir, cleanName)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return err
			}
			mode := os.FileMode(header.Mode)
			if runtime.GOOS == "windows" {
				mode = 0o644
			}
			output, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
			if err != nil {
				return err
			}
			if _, err := io.Copy(output, tarReader); err != nil {
				_ = output.Close()
				return err
			}
			if err := output.Close(); err != nil {
				return err
			}
		}
	}
}
