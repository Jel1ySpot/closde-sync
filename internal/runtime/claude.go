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

type claudePackage struct {
	NpmName    string
	InstallDir string
	CLIPath    string
}

var (
	packageCC  = claudePackage{NpmName: "@anthropic-ai/claude-code", InstallDir: "claude", CLIPath: "cli.js"}
	packageCCB = claudePackage{NpmName: "claude-code-best", InstallDir: "claude-code-best", CLIPath: "dist/cli.js"}
)

// ResolveClaudePackage parses CLOSDE_CLAUDE_VERSION into a package and an
// optional pinned version (empty means "latest"). Accepted forms:
//
//	""            -> claude-code, latest
//	"cc"          -> claude-code, latest
//	"cc:<v>"      -> claude-code, version <v>
//	"ccb"         -> claude-code-best, latest
//	"ccb:<v>"     -> claude-code-best, version <v>
//	"<v>"         -> claude-code, version <v>   (backward compatibility)
func ResolveClaudePackage(raw string) (claudePackage, string) {
	raw = strings.TrimSpace(raw)
	switch {
	case raw == "" || raw == "cc":
		return packageCC, ""
	case raw == "ccb":
		return packageCCB, ""
	case strings.HasPrefix(raw, "cc:"):
		return packageCC, strings.TrimPrefix(raw, "cc:")
	case strings.HasPrefix(raw, "ccb:"):
		return packageCCB, strings.TrimPrefix(raw, "ccb:")
	default:
		return packageCC, raw
	}
}

func EnsureClaudeCode(cfg Config) error {
	if FileExists(cfg.ClaudeCLIPath) {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(cfg.ClaudeRootPath), 0o755); err != nil {
		return err
	}

	archiveURL := packageTarballURL(cfg.ClaudeNpmPackage, cfg.ClaudeVersion)
	response, err := http.Get(archiveURL)
	if err != nil {
		return fmt.Errorf("download %s %s: %w", cfg.ClaudeNpmPackage, cfg.ClaudeVersion, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s %s: unexpected status %s", cfg.ClaudeNpmPackage, cfg.ClaudeVersion, response.Status)
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
		return fmt.Errorf("download %s %s: extracted package directory missing", cfg.ClaudeNpmPackage, cfg.ClaudeVersion)
	}

	return CopyDir(extractedPackagePath, cfg.ClaudeRootPath)
}

func FetchLatestNpmVersion(npmPackage string) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	response, err := client.Get("https://registry.npmjs.org/" + npmPackage)
	if err != nil {
		return "", fmt.Errorf("query %s registry metadata: %w", npmPackage, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("query %s registry metadata: unexpected status %s", npmPackage, response.Status)
	}

	var payload struct {
		DistTags map[string]string `json:"dist-tags"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode %s registry metadata: %w", npmPackage, err)
	}

	latestVersion := strings.TrimSpace(payload.DistTags["latest"])
	if latestVersion == "" {
		return "", fmt.Errorf("missing latest dist-tag for %s", npmPackage)
	}

	return latestVersion, nil
}

func packageTarballURL(npmPackage, version string) string {
	base := npmPackage
	if idx := strings.LastIndex(base, "/"); idx >= 0 {
		base = base[idx+1:]
	}
	return fmt.Sprintf("https://registry.npmjs.org/%s/-/%s-%s.tgz", npmPackage, base, version)
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
