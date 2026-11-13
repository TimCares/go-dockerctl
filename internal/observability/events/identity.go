package events

import (
	"context"
	"fmt"

	"github.com/TimCares/go-dockerctl/internal/identity"
	"github.com/TimCares/go-see"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// IdentityCreatedID identifies [IdentityCreated] events.
var IdentityCreatedID = see.NewID("identity.created")

var identitiesCreatedTotal, _ = meter.Int64Counter("identities_created_total")

// IdentityCreated records that a new identity was created using age.
type IdentityCreated struct {
	Hostname     string `json:"host"`
	IdentityPath string `json:"identityPath"`
	Recipient    string `json:"recipient"`
}

// ID implements [see.Event].
func (IdentityCreated) ID() see.ID { return IdentityCreatedID }

// Level implements [see.Leveler].
func (e IdentityCreated) Level() see.Level {
	return see.LevelInfo
}

// Measure increments a total identities created counter.
//
// Backend is always age.
func (e IdentityCreated) Measure(ctx context.Context) {
	identitiesCreatedTotal.Add(
		ctx,
		1,
		metric.WithAttributes(
			attribute.String("hostname", e.Hostname),
		),
	)
}

// Message is the log message, and printed to the terminal with the console log format.
func (e IdentityCreated) Message() string {
	return fmt.Sprintf("Created a new age identity at %s. Add as recipient using \"identity add <recipient>\".", e.IdentityPath)
}

// ---

// IdentityAddedID identifies [IdentityAdded] events.
var IdentityAddedID = see.NewID("identity.added")

var identitiesAddedTotal, _ = meter.Int64Counter("identities_added_total")

// IdentityAdded records that one identity was added as a recipient to the secrets.
//
// Here we explicitly have "Env" as a field.
// This module is the only exception where we include "domain knowledge" of other modules.
type IdentityAdded struct {
	Hostname  string                  `json:"host"`
	Env       string                  `json:"env"`
	KeySource identity.SOPSKeySources `json:"keySource"`
	Recipient string                  `json:"recipient"`
}

// ID implements [see.Event].
func (IdentityAdded) ID() see.ID { return IdentityAddedID }

// Level reports the log level. An identity having been added is always an info.
func (e IdentityAdded) Level() see.Level {
	return see.LevelInfo
}

// Measure increments a total identities added counter with "BackendType" as the label.
func (e IdentityAdded) Measure(ctx context.Context) {
	identitiesAddedTotal.Add(
		ctx,
		1,
		metric.WithAttributes(
			attribute.String("hostname", e.Hostname),
			attribute.String("key_source", string(e.KeySource)),
		),
	)
}
