package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	closdelog "closde-sync/internal/logging"
	closderuntime "closde-sync/internal/runtime"
	"closde-sync/internal/xray"
	"closde-sync/server"
)

const helpText = `closde

Usage:
  closde [client flags...]
  closde-server [server flags]
  closde-server create-token [flags] [name]
  closde-server get-config [flags]

Modes:
  Binary names containing "server" run in server mode.
  Other binary names run in client mode and forward all args to Node.

Environment:
  ~/.closde/.env is loaded automatically when present.
  Optional: CLOSDE_CLAUDE_VERSION, CLOSDE_PROXY, CLOSDE_PRELOAD_FILE, DEBUG_MODE
`
var AppVersion = "dev"

func Run(argv []string) error {
	if len(argv) == 0 {
		return runClientMode(nil)
	}
	if isServerBinary(argv[0]) {
		return runServerMode(argv[1:])
	}
	return runClientMode(argv[1:])
}

func PrintHelp() {
	fmt.Print(helpText)
}

func runServerMode(args []string) error {
	if len(args) == 0 {
		return server.Run(nil)
	}

	switch args[0] {
	case "create-token", "get-config":
		return server.RunCommand(args[0], args[1:])
	case "serve":
		return server.Run(args[1:])
	case "help", "-h", "--help":
		PrintHelp()
		return nil
	default:
		return server.Run(args)
	}
}

func runClientMode(args []string) error {
	cfg, err := closderuntime.LoadConfig(AppVersion)
	if err != nil {
		return err
	}

	closdelog.ConfigureFromEnv()
	closdelog.With("cli").Debug("starting client mode", "binary", os.Args[0], "args", args)

	if err := closderuntime.EnsureClaudeCode(cfg); err != nil {
		return err
	}

	var proxy *xray.Instance
	if strings.TrimSpace(cfg.ProxyURI) != "" {
		proxy, err = xray.StartProxy(cfg.ProxyURI, cfg.ProxyHost, cfg.ProxyPort, closdelog.IsDebugMode())
		if err != nil {
			return err
		}
		defer func() {
			_ = proxy.Close()
		}()
	}

	nodePath, err := exec.LookPath("node")
	if err != nil {
		return errors.New("node is not installed or not available in PATH")
	}

	commandArgs := append([]string{cfg.ClaudeCLIPath}, args...)
	closdelog.With("cli").Debug("launching node client", "node", nodePath, "args", commandArgs)

	command := exec.Command(nodePath, commandArgs...)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	command.Env = closderuntime.PrepareCommandEnv(cfg, proxy != nil)
	return command.Run()
}

func isServerBinary(executablePath string) bool {
	name := strings.ToLower(filepath.Base(executablePath))
	return strings.Contains(name, "server")
}
