package runtime

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	DefaultProxyHost = "127.0.0.1"
	DefaultProxyPort = 11808
)

type Config struct {
	DataDir          string
	AppVersion       string
	ClaudeVersion    string
	ClaudeNpmPackage string
	ProxyURI         string
	ProxyHost        string
	ProxyPort        int
	PreloadJSPath    string
	ClaudeCLIPath    string
	ClaudeRootPath   string
	LocalClaude      bool
}

func LoadConfig(appVersion string) (Config, error) {
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		return Config{}, err
	}

	dataDir := filepath.Join(homeDirectory, ".closde")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return Config{}, err
	}

	envFilePath := filepath.Join(dataDir, ".env")
	if err := ensureEnvFile(envFilePath); err != nil {
		return Config{}, err
	}
	if err := LoadEnvFile(envFilePath); err != nil {
		return Config{}, err
	}

	rawClaudeVersion := strings.TrimSpace(os.Getenv("CLOSDE_CLAUDE_VERSION"))
	if rawClaudeVersion == "local" {
		return Config{
			DataDir:          dataDir,
			AppVersion:       appVersion,
			ClaudeVersion:    "local",
			ClaudeNpmPackage: "",
			ProxyURI:         strings.TrimSpace(os.Getenv("CLOSDE_PROXY")),
			ProxyHost:        DefaultProxyHost,
			ProxyPort:        DefaultProxyPort,
			PreloadJSPath:    "",
			ClaudeCLIPath:    "claude",
			ClaudeRootPath:   "",
			LocalClaude:      true,
		}, nil
	}

	pkg, claudeVersion := ResolveClaudePackage(rawClaudeVersion)
	if claudeVersion == "" {
		resolvedVersion, resolveErr := FetchLatestNpmVersion(pkg.NpmName)
		if resolveErr != nil {
			return Config{}, fmt.Errorf("CLOSDE_CLAUDE_VERSION not pinned and failed to resolve latest %s version: %w", pkg.NpmName, resolveErr)
		}
		claudeVersion = resolvedVersion
	}

	claudeRootPath := filepath.Join(dataDir, pkg.InstallDir, claudeVersion)

	preloadJSPath, err := ResolvePreloadFile(dataDir, appVersion)
	if err != nil {
		return Config{}, err
	}

	return Config{
		DataDir:          dataDir,
		AppVersion:       appVersion,
		ClaudeVersion:    claudeVersion,
		ClaudeNpmPackage: pkg.NpmName,
		ProxyURI:         strings.TrimSpace(os.Getenv("CLOSDE_PROXY")),
		ProxyHost:        DefaultProxyHost,
		ProxyPort:        DefaultProxyPort,
		PreloadJSPath:    preloadJSPath,
		ClaudeCLIPath:    filepath.Join(claudeRootPath, pkg.CLIPath),
		ClaudeRootPath:   claudeRootPath,
		LocalClaude:      false,
	}, nil
}

func PrepareCommandEnv(cfg Config, proxyEnabled bool) []string {
	env := os.Environ()
	setEnv(&env, "DISABLE_TELEMETRY", "1")
	setEnv(&env, "NODE_USE_ENV_PROXY", "1")
	if !cfg.LocalClaude {
		setEnv(&env, "NODE_OPTIONS", mergeNodeOptions(os.Getenv("NODE_OPTIONS"), "--require "+cfg.PreloadJSPath))
	}

	if proxyEnabled {
		proxyURL := fmt.Sprintf("http://%s:%d", cfg.ProxyHost, cfg.ProxyPort)
		setEnv(&env, "HTTP_PROXY", proxyURL)
		setEnv(&env, "HTTPS_PROXY", proxyURL)
	}

	return env
}

func ensureEnvFile(filePath string) error {
	if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
		if writeErr := os.WriteFile(filePath, []byte(""), 0o644); writeErr != nil {
			return fmt.Errorf("create %s: %w", filePath, writeErr)
		}
		return fmt.Errorf("created %s; optionally set CLOSDE_CLAUDE_VERSION to pin a version", filePath)
	} else if err != nil {
		return err
	}

	return nil
}

func mergeNodeOptions(existing string, required string) string {
	existing = strings.TrimSpace(existing)
	required = strings.TrimSpace(required)
	if existing == "" {
		return required
	}
	if strings.Contains(existing, required) {
		return existing
	}
	return existing + " " + required
}

func setEnv(env *[]string, key string, value string) {
	prefix := key + "="
	for index, entry := range *env {
		if strings.HasPrefix(entry, prefix) {
			(*env)[index] = prefix + value
			return
		}
	}
	*env = append(*env, prefix+value)
}
