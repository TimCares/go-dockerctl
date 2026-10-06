package events

import (
	"context"
	"fmt"

	"github.com/TimCares/go-see"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// IdentityCreatedID identifies [IdentityCreated] events.
var IdentityCreatedID = see.NewID("identity.created")

var identitiesCreatedTotal, _ = meter.Int64Counter("identities_created_total")

// IdentityCreated records that a new identity was created using age.
type IdentityCreated struct {
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
	identitiesCreatedTotal.Add(ctx, 1)
}

// Message is the log message, and printed to the terminal with the console log format.
func (e IdentityCreated) Message() string {
	return fmt.Sprintf("Created a new age identity at %s. Add its recipient to your .sops.yaml.", e.IdentityPath)
}

// ---

// IdentityAddedID identifies [IdentityAdded] events.
var IdentityAddedID = see.NewID("identity.added")

// SOPSIdentityBackendTypes are the kinds of key SOPS can encrypt secrets for.
type SOPSIdentityBackendTypes string

const (
	// Age is an age X25519 key.
	Age SOPSIdentityBackendTypes = "age"
	// PGP is a PGP / GnuPG key.
	PGP SOPSIdentityBackendTypes = "pgp"
	// KMS is any Key Management System, e.g. HashiCorp Vault.
	KMS SOPSIdentityBackendTypes = "kms"
)

var identitiesAddedTotal, _ = meter.Int64Counter("identities_added_total")

// IdentityAdded records that one identity was added as a recipient to the secrets.
type IdentityAdded struct {
	CreationRule string                   `json:"creationRule"`
	BackendType  SOPSIdentityBackendTypes `json:"backendType"`
	IdentityPath string                   `json:"identityPath"`
	Recipient    string                   `json:"recipient"`
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
			attribute.String("backend_type", string(e.BackendType)),
		),
	)
}
