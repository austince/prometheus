package xds

import (
	"context"
	"github.com/go-kit/kit/log"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"google.golang.org/grpc"
)

type GrpcDiscovery struct {
	server string
	log log.Logger
}

type GrpcClient struct {
	conn *grpc.ClientConn
}

func newGrpcDiscovery(conf *SDConfig, logger log.Logger) (*GrpcDiscovery, error) {
	d := &GrpcDiscovery{
		server: conf.Server,
	}

	return d, nil
}

func (d *GrpcDiscovery) Run(ctx context.Context, up chan<- []*targetgroup.Group) {
	d.log.Log("msg", "Running GRPC Discovery")
	// Must sync the latest groups, which should be buffered in memory?
}
