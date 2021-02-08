package xds

import (
	"errors"
	"fmt"
	"github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery"
	"net/url"
	"time"
)

type DiscoveryMode string

const (
	GRPCMode = DiscoveryMode("grpc")
	HTTPMode = DiscoveryMode("http")
)

type HTTPConfig struct {
	config.HTTPClientConfig `yaml:",inline"`
	RefreshInterval         model.Duration `yaml:"refresh_interval,omitempty"`
}

type GRPCConfig struct {
}

// DefaultSDConfig is the default xDS SD configuration.
var DefaultSDConfig = SDConfig{
	Http: &HTTPConfig{
		RefreshInterval: model.Duration(30 * time.Second),
	},
}

// TODO: how to support different API versions?
type SDConfig struct {
	mode   DiscoveryMode
	Server string        `yaml:"server,omitempty"`
	Http   *HTTPConfig   `yaml:"http,omitempty"`
	Grpc   *GRPCConfig   `yaml:"grpc,omitempty"`
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (c *SDConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*c = DefaultSDConfig
	type plain SDConfig
	err := unmarshal((*plain)(c))
	if err != nil {
		return err
	}
	if len(c.Server) == 0 {
		return errors.New("xds_sd: empty or null xDS server")
	}
	parsedUrl, err := url.Parse(c.Server)
	if err != nil {
		return err
	}

	if len(parsedUrl.Scheme) == 0 || len(parsedUrl.Host) == 0 {
		return errors.New("xds_sd: invalid xDS server URL")
	}

	switch parsedUrl.Scheme {
	case "grpc":
	case "grpcs":
		c.mode = GRPCMode
		return nil
	case "http":
	case "https":
		c.mode = HTTPMode
		return c.Http.Validate()
	default:
		return  fmt.Errorf("unsupported server protocol %s, must be either 'grpc'/'grpcs' or 'http'/'https'", parsedUrl.Scheme)
	}

	return nil
}

func init() {
	discovery.RegisterConfig(&SDConfig{})
}

func (c *SDConfig) Name() string {
	return string("xds-" + c.mode)
}

func (c *SDConfig) NewDiscoverer(opts discovery.DiscovererOptions) (discovery.Discoverer, error) {
	switch c.mode {
	case GRPCMode:
		return newGrpcDiscovery(c, opts.Logger)
	case HTTPMode:
		return newHttpDiscovery(c, opts.Logger)
	default:
		return nil, fmt.Errorf("invalid mode %s, must be either 'grpc' or 'http'", c.mode)
	}
}
