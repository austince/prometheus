package xds

import (
	"context"
	"github.com/go-kit/kit/log"
	"github.com/prometheus/common/config"
	"github.com/prometheus/prometheus/discovery/refresh"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"net/http"
	"time"
)

type HTTPDiscovery struct {
	server string
	*refresh.Discovery
	client *http.Client
}

func newHttpDiscovery(conf *SDConfig, logger log.Logger) (*HTTPDiscovery, error) {
	rt, err := config.NewRoundTripperFromConfig(conf.Http.HTTPClientConfig, "xds_sd", false, false)
	if err != nil {
		return nil, err
	}

	d := &HTTPDiscovery{
		client: &http.Client{Transport: rt},
		server: conf.Server,
	}
	d.Discovery = refresh.NewDiscovery(
		logger,
		"xds",
		time.Duration(conf.Http.RefreshInterval),
		d.refresh,
	)
	return d, nil
}

func (d *HTTPDiscovery) Run(ctx context.Context, up chan<- []*targetgroup.Group) {

}

func (d *HTTPDiscovery) refresh(ctx context.Context) ([]*targetgroup.Group, error) {
	return nil, nil
}
