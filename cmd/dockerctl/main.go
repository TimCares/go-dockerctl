// Dockerctl manages multiple docker compose projects with SOPS encrypted secrets.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/TimCares/go-dockerctl/internal/cli"
	"github.com/TimCares/go-dockerctl/internal/observability"
)

func main() {
	os.Exit(run())
}

// run exists so deferred calls still happen -> os.Exit skips.
func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	err := cli.New().Run(ctx, os.Args)

	// Not derived from ctx: after Ctrl-C it is cancelled and the flush would abort.
	// The deadline keeps an unreachable collector from delaying exit.
	shutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	shutErr := observability.Shutdown(shutCtx)

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
	}
	if shutErr != nil {
		fmt.Fprintln(os.Stderr, "Error:", shutErr)
	}
	if err != nil || shutErr != nil {
		return 1
	}
	return 0
}
