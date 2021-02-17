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
	// metaLabelPrefix is the meta prefix used for all meta labels.
	// in this discovery.
	metaLabelPrefix = model.MetaLabelPrefix + "xds_"
	apiVersionLabel      = metaLabelPrefix + "api_version"
	serverLabel          = metaLabelPrefix + "server"
	protocolVersionLabel = metaLabelPrefix + "protocol_version"
	modeLabel = metaLabelPrefix + "mode"

	source = "xds"
)

const (
	GRPCMode = DiscoveryMode("grpc")
	HTTPMode = DiscoveryMode("http")
)

// The xDS protocol version
type ProtocolVersion string

const (
	ProtocolV3 = ProtocolVersion("v3")
)

type ApiVersion string

const (
	V1Alpha1 = ApiVersion("v1alpha1")
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

type SDConfig struct {
	Mode            DiscoveryMode   // set from server protocol
	Server          string          `yaml:"server,omitempty"`
	Http            *HTTPConfig     `yaml:"http,omitempty"`
	Grpc            *GRPCConfig     `yaml:"grpc,omitempty"`
	ProtocolVersion ProtocolVersion `yaml:"protocolVersionLabel"`
	ApiVersion      ApiVersion      `yaml:"apiVersionLabel"`
}

func validateProtocolVersion(version ProtocolVersion) error {
	switch version {
	case ProtocolV3:
		return nil
	default:
		return fmt.Errorf("unsupported xDS protocol version %s. Only v3 is supported", version)
	}
}

func validateApiVersion(version ApiVersion) error {
	switch version {
	case V1Alpha1:
		return nil
	default:
		return fmt.Errorf("unsupported apiVersionLabel %s. Only v1alpha1 is supported", version)
	}
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (c *SDConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*c = DefaultSDConfig
	type plain SDConfig
	err := unmarshal((*plain)(c))
	if err != nil {
		return err
	}

	if err = validateProtocolVersion(c.ProtocolVersion); err != nil {
		return err
	}

	if err = validateApiVersion(c.ApiVersion); err != nil {
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
		c.Mode = GRPCMode
		return nil
	case "http":
	case "https":
		c.Mode = HTTPMode
		return c.Http.Validate()
	default:
		return fmt.Errorf("unsupported server protocol %s, must be either 'grpc'/'grpcs' or 'http'/'https'", parsedUrl.Scheme)
	}

	return nil
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
		return newGrpcDiscovery(c, opts.Logger)
	case HTTPMode:
		return newHttpDiscovery(c, opts.Logger)
	default:
		return nil, fmt.Errorf("invalid Mode %s, must be either 'grpc' or 'http'", c.Mode)
	}
}
