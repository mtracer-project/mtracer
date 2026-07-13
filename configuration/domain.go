package configuration

import (
	"context"
	"fmt"

	"github.com/mtracer-project/mtracer/configuration/jaeger"
	"github.com/mtracer-project/mtracer/configuration/openobserve"
	"github.com/mtracer-project/mtracer/trace"
)

type AppConfig struct {
	Directory         string                         `mapstructure:"directory"`
	Verbose           bool                           `mapstructure:"verbose"`
	BackendType       string                         `mapstructure:"backend_type"`
	Quiet             bool                           `mapstructure:"quiet"`
	OpenObserveConfig *openobserve.OpenObserveConfig `mapstructure:"openobserve"`
	JaegerConfig      *jaeger.JaegerConfig           `mapstructure:"jaeger"`
}

func (c *AppConfig) NewTraceAdapterFromConfig(ctx context.Context) (trace.TraceAdapter, error) {
	switch c.BackendType {
	case "openobserve":
		repo := openobserve.NewOpenObserveTraceRepository(c.OpenObserveConfig, ctx)
		return openobserve.NewOpenObserveTraceAdapter(repo)
	case "jaeger":
		repo := jaeger.NewJaegerTraceRepository(c.JaegerConfig, ctx)
		return jaeger.NewJaegerTraceAdapter(repo)
	default:
		return nil, fmt.Errorf("unsupported backend type: %s", c.BackendType)
	}
}
