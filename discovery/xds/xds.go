package xds

import (
	"context"
	"fmt"
	"github.com/prometheus/common/config"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/discovery/refresh"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"log"
	"net/http"
)

type GRPCDiscovery struct {
	log log.Logger
}

func (d *GRPCDiscovery) Run(ctx context.Context, up chan<- []*targetgroup.Group) {
	d.log.Print("Running GRPC Discovery")
	// Must sync the latest groups, which should be buffered in memory?
}

type HTTPDiscovery struct {
	*refresh.Discovery
	client *http.Client
}

func (d *HTTPDiscovery) Run(ctx context.Context, up chan<- []*targetgroup.Group) {

}

type DiscoveryMode string

const (
	GRPCMode = DiscoveryMode("grpc")
	HTTPMode = DiscoveryMode("http")
)

type SDConfig struct {
	Mode DiscoveryMode `yaml:"mode"`
	HTTPClientConfig config.HTTPClientConfig `yaml:"http,omitempty"`
}

func init() {
	discovery.RegisterConfig(&SDConfig{})
}

func (c *SDConfig) Name() string {
	return string("xds-" + c.Mode)
}

func (c *SDConfig) NewDiscoverer(opts discovery.DiscovererOptions) (discovery.Discoverer, error) {
	switch c.Mode {
	case GRPCMode:
		return &GRPCDiscovery{}, nil
	case HTTPMode:
		// TODO: configure refresh
		return &HTTPDiscovery{}, nil
	default:
		return nil, fmt.Errorf("invalid mode %s, must be either 'grpc' or 'http'", c.Mode)
	}
}
