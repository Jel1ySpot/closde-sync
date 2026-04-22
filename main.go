package main

import (
	"os"

	"closde-sync/internal/cli"
	closdelog "closde-sync/internal/logging"
)

func main() {
	closdelog.ConfigureFromEnv()

	if err := cli.Run(os.Args); err != nil {
		closdelog.With("main").Error("closde failed", "error", err)
		os.Exit(1)
	}
}
