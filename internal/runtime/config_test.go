package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigLocalClaude(t *testing.T) {
	homeDir := t.TempDir()
	dataDir := filepath.Join(homeDir, ".closde")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, ".env"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", homeDir)
	t.Setenv("CLOSDE_CLAUDE_VERSION", "local")

	cfg, err := LoadConfig("test-version")
	if err != nil {
		t.Fatal(err)
	}

	if !cfg.LocalClaude {
		t.Fatal("LocalClaude = false, want true")
	}
	if cfg.ClaudeCLIPath != "claude" {
		t.Fatalf("ClaudeCLIPath = %q, want %q", cfg.ClaudeCLIPath, "claude")
	}
	if cfg.PreloadJSPath != "" {
		t.Fatalf("PreloadJSPath = %q, want empty", cfg.PreloadJSPath)
	}
}

func TestPrepareCommandEnvLocalClaudeDoesNotAddNodeOptions(t *testing.T) {
	unsetEnvForTest(t, "NODE_OPTIONS")

	env := PrepareCommandEnv(Config{LocalClaude: true}, false)
	if nodeOptions, ok := lookupEnv(env, "NODE_OPTIONS"); ok {
		t.Fatalf("NODE_OPTIONS = %q, want absent", nodeOptions)
	}
}

func TestPrepareCommandEnvLocalClaudePreservesExistingNodeOptions(t *testing.T) {
	t.Setenv("NODE_OPTIONS", "--inspect")

	env := PrepareCommandEnv(Config{LocalClaude: true}, false)
	nodeOptions, ok := lookupEnv(env, "NODE_OPTIONS")
	if !ok {
		t.Fatal("NODE_OPTIONS missing from inherited environment")
	}
	if strings.Contains(nodeOptions, "--require") {
		t.Fatalf("NODE_OPTIONS = %q, want no preload require", nodeOptions)
	}
	if nodeOptions != "--inspect" {
		t.Fatalf("NODE_OPTIONS = %q, want inherited value unchanged", nodeOptions)
	}
}

func unsetEnvForTest(t *testing.T, key string) {
	t.Helper()

	value, ok := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if ok {
			_ = os.Setenv(key, value)
			return
		}
		_ = os.Unsetenv(key)
	})
}

func TestPrepareCommandEnvManagedClaudeAddsPreload(t *testing.T) {
	t.Setenv("NODE_OPTIONS", "--inspect")

	env := PrepareCommandEnv(Config{PreloadJSPath: "/tmp/preload.js"}, false)
	nodeOptions, ok := lookupEnv(env, "NODE_OPTIONS")
	if !ok {
		t.Fatal("NODE_OPTIONS missing")
	}
	if !strings.Contains(nodeOptions, "--require /tmp/preload.js") {
		t.Fatalf("NODE_OPTIONS = %q, want preload require", nodeOptions)
	}
}

func lookupEnv(env []string, key string) (string, bool) {
	prefix := key + "="
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			return strings.TrimPrefix(entry, prefix), true
		}
	}
	return "", false
}
