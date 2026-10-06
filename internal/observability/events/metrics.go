package events

import (
	"go.opentelemetry.io/otel"
)

const dockerCTLMeterIdentifier = "github.com/TimCares/go-dockerctl"

// meter creates the instruments of every event in this package.
var meter = otel.Meter(dockerCTLMeterIdentifier)
