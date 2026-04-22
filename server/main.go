package server

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

const fileWatchPollInterval = 250 * time.Millisecond

func Run(args []string) error {
	cfg, err := parseServeConfig(args)
	if err != nil {
		return err
	}
	return Serve(cfg)
}

func RunCommand(command string, args []string) error {
	switch command {
	case "create-token":
		return runCreateToken(args)
	case "get-config":
		return runGetConfig(args)
	default:
		return fmt.Errorf("unsupported server command: %s", command)
	}
}

func Serve(cfg Config) error {
	app, err := newApp(cfg)
	if err != nil {
		return err
	}

	if cfg.WatchLocalState {
		app.watchLocalState()
	}

	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           app.routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	logger().Info("server listening", "addr", cfg.Addr)
	return httpServer.ListenAndServe()
}

func runCreateToken(args []string) error {
	cfg, err := parseCreateTokenConfig(args)
	if err != nil {
		return err
	}

	store := newStateStore(cfg.ConfigFile)
	token, err := store.AddToken(cfg.Token, cfg.Name)
	if err != nil {
		return err
	}

	fmt.Println(token)
	return nil
}

func runGetConfig(args []string) error {
	cfg, err := parseGetConfig(args)
	if err != nil {
		return err
	}

	store := newStateStore(cfg.ConfigFile)
	if _, _, err := store.SyncStateFromClaudeFiles(cfg.ClaudeSettingsFile, cfg.ClaudeCredentialsFile); err != nil {
		return err
	}

	fmt.Println(cfg.ConfigFile)
	return nil
}

func waitForFileChange(paths ...string) {
	files := make(map[string]fileSnapshot, len(paths))
	for _, path := range paths {
		files[path] = snapshotFile(path)
	}

	for {
		time.Sleep(fileWatchPollInterval)
		for _, path := range paths {
			next := snapshotFile(path)
			if next == files[path] {
				continue
			}
			files[path] = next
			return
		}
	}
}

type fileSnapshot struct {
	exists bool
	size   int64
	mtime  time.Time
}

func snapshotFile(path string) fileSnapshot {
	info, err := os.Stat(path)
	if err != nil {
		return fileSnapshot{}
	}

	return fileSnapshot{exists: true, size: info.Size(), mtime: info.ModTime()}
}
