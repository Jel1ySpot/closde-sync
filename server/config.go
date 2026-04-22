package server

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Addr                  string
	ConfigFile            string
	WatchLocalState       bool
	ClaudeSettingsFile    string
	ClaudeCredentialsFile string
}

type createTokenConfig struct {
	ConfigFile string
	Token      string
	Name       string
}

type getConfigCommandConfig struct {
	ConfigFile            string
	ClaudeSettingsFile    string
	ClaudeCredentialsFile string
}

func parseServeConfig(args []string) (Config, error) {
	flags := flag.NewFlagSet("closde-server", flag.ContinueOnError)
	addr := flags.String("addr", envOrDefault("CLOSDE_SERVER_ADDR", ":8080"), "HTTP listen address")
	configFile := flags.String("config-file", envOrDefault("CLOSDE_SERVER_CONFIG_FILE", defaultConfigFilePath()), "Server config file path")
	watchLocalState := flags.Bool("watch-local-state", false, "Watch local Claude state files and broadcast state updates")
	claudeSettingsFile := flags.String("claude-settings", defaultClaudeSettingsPath(), "Claude settings file path")
	claudeCredentialsFile := flags.String("claude-credentials", defaultClaudeCredentialsPath(), "Claude credentials file path")
	if err := flags.Parse(args); err != nil {
		return Config{}, err
	}

	return Config{
		Addr:                  *addr,
		ConfigFile:            *configFile,
		WatchLocalState:       *watchLocalState,
		ClaudeSettingsFile:    *claudeSettingsFile,
		ClaudeCredentialsFile: *claudeCredentialsFile,
	}, nil
}

func parseCreateTokenConfig(args []string) (createTokenConfig, error) {
	flags := flag.NewFlagSet("create-token", flag.ContinueOnError)
	configFile := flags.String("config-file", envOrDefault("CLOSDE_SERVER_CONFIG_FILE", defaultConfigFilePath()), "Server config file path")
	token := flags.String("token", "", "Token to persist; when empty a random token is generated")
	if err := flags.Parse(args); err != nil {
		return createTokenConfig{}, err
	}

	name := ""
	if remaining := flags.Args(); len(remaining) > 0 {
		name = strings.TrimSpace(remaining[0])
	}

	return createTokenConfig{ConfigFile: *configFile, Token: strings.TrimSpace(*token), Name: name}, nil
}

func parseGetConfig(args []string) (getConfigCommandConfig, error) {
	flags := flag.NewFlagSet("get-config", flag.ContinueOnError)
	configFile := flags.String("config-file", envOrDefault("CLOSDE_SERVER_CONFIG_FILE", defaultConfigFilePath()), "Server config file path")
	claudeSettingsFile := flags.String("claude-settings", defaultClaudeSettingsPath(), "Claude settings file path")
	claudeCredentialsFile := flags.String("claude-credentials", defaultClaudeCredentialsPath(), "Claude credentials file path")
	if err := flags.Parse(args); err != nil {
		return getConfigCommandConfig{}, err
	}

	return getConfigCommandConfig{
		ConfigFile:            *configFile,
		ClaudeSettingsFile:    *claudeSettingsFile,
		ClaudeCredentialsFile: *claudeCredentialsFile,
	}, nil
}

func envOrDefault(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func defaultConfigFilePath() string {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return "config.json"
	}
	return filepath.Join(workingDirectory, "config.json")
}

func defaultClaudeSettingsPath() string {
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("~", ".claude.json")
	}
	return filepath.Join(homeDirectory, ".claude.json")
}

func defaultClaudeCredentialsPath() string {
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("~", ".claude", ".credentials.json")
	}
	return filepath.Join(homeDirectory, ".claude", ".credentials.json")
}
