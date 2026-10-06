package config

import (
	"context"
	"fmt"
)

// We key [Config] in [context.Context] on this private type.
type ctxConfigKey struct{}

// ContextWithConfig adds [Config] as a value to a [context.Context].
func ContextWithConfig(ctx context.Context, config Config) context.Context {
	return context.WithValue(ctx, ctxConfigKey{}, config)
}

// FromContext extracts [Config] from [context.Context].
func FromContext(ctx context.Context) (*Config, error) {
	config, ok := ctx.Value(ctxConfigKey{}).(Config)
	if !ok {
		return nil, fmt.Errorf("config missing from ctx")
	}
	return &config, nil
}
