// Package observability contains event, logging and metric definitions for dockerctl.
package observability

import (
	"context"

	"github.com/TimCares/go-see"

	"github.com/TimCares/go-dockerctl/internal/config"
)

// Init sets up logging, makes it the default [see.Emitter], and starts metrics if enabled.
func Init(ctx context.Context, config config.ObservabilityConfig) error {
	logger, err := InitLogging(ctx, config.Logging)
	if err != nil {
		return err
	}

	defaultEmitter, err := see.New(see.Config{Logger: logger})
	if err != nil {
		return err
	}
	see.SetDefault(defaultEmitter)

	if config.Otel.Enabled {
		if err := InitMetrics(ctx, config.Otel); err != nil {
			return err
		}
	}

	return nil
}

// Shutdown flushes metrics and the log file. Safe to call when [Init] did not run.
func Shutdown(ctx context.Context) error {
	err := shutdownMetrics(ctx)
	FlushAndSyncLogFile()
	return err
}
