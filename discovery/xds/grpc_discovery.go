package xds

import (
	"context"
	"github.com/go-kit/kit/log"
	"github.com/prometheus/prometheus/discovery/targetgroup"
)

type GRPCDiscovery struct {
	server string
	log log.Logger
}

func newGrpcDiscovery(conf *SDConfig, logger log.Logger) (*GRPCDiscovery, error) {
	d := &GRPCDiscovery{
		server: conf.Server,
	}

	return d, nil
}

func (d *GRPCDiscovery) Run(ctx context.Context, up chan<- []*targetgroup.Group) {
	d.log.Log("msg", "Running GRPC Discovery")
	// Must sync the latest groups, which should be buffered in memory?
}
