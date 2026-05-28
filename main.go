package main

import (
	"os"

	"closde-sync/internal/cli"
	closdelog "closde-sync/internal/logging"
)

var Version = "dev"

func main() {
	closdelog.ConfigureFromEnv()
	cli.AppVersion = Version

	if err := cli.Run(os.Args); err != nil {
		closdelog.With("main").Error("closde failed", "error", err)
		os.Exit(1)
	}
}
