// Dockerctl manages multiple docker compose projects with SOPS encrypted secrets.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/TimCares/go-dockerctl/internal/cli"
	"github.com/TimCares/go-dockerctl/internal/logger"
)

func main() {
	os.Exit(run())
}

// run exists so deferred calls still happen -> os.Exit skips.
func run() int {
	defer logger.Sync()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := cli.New().Run(ctx, os.Args); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return 1
	}

	return 0
}
