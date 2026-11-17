package events

import (
	"context"

	"github.com/TimCares/go-dockerctl/internal/identity"
	"github.com/TimCares/go-see"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// ---

// IdentityAddedID identifies [IdentityAdded] events.
var IdentityAddedID = see.NewID("identity.added")

var identitiesAddedTotal, _ = meter.Int64Counter("identities_added_total")

// IdentityAdded records that one identity was added as a recipient to the secrets of one env.
type IdentityAdded struct {
	KeySource identity.SOPSKeySources `json:"keySource"`
	Recipient string                  `json:"recipient"`
}

// ID implements [see.Event].
func (IdentityAdded) ID() see.ID { return IdentityAddedID }

// Level reports the log level. An identity having been added is always an info.
func (e IdentityAdded) Level() see.Level {
	return see.LevelInfo
}

// Measure increments a total identities added counter with "key_source" as the label.
func (e IdentityAdded) Measure(ctx context.Context) {
	identitiesAddedTotal.Add(
		ctx,
		1,
		metric.WithAttributes(
			attribute.String("key_source", string(e.KeySource)),
		),
	)
}
