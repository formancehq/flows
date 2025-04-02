package tracer

import (
	"go.opentelemetry.io/otel"
)

var Tracer = otel.Tracer("runner")
