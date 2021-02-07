// Copyright 2021 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package xds

import (
	"encoding/json"
	"fmt"
	"github.com/go-kit/kit/log"
	"net/url"
	"time"

	"github.com/golang/protobuf/ptypes/any"
	"github.com/pkg/errors"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/discovery/refresh"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/prometheus/prometheus/util/strutil"
)

var (
	// DefaultKumaSDConfig is the default Kuma MADS SD configuration.
	DefaultKumaSDConfig = KumaSDConfig{
		xdsSDConfig: xdsSDConfig{
			HTTP: &HTTPConfig{
				RefreshInterval: model.Duration(30 * time.Second),
			},
		},
	}
)

const (
	// kumaMetaLabelPrefix is the meta prefix used for all kuma meta labels.
	// in this discovery.
	kumaMetaLabelPrefix = model.MetaLabelPrefix + "kuma_"

	// meshLabel is the name of the label that holds the mesh name.
	meshLabel = kumaMetaLabelPrefix + "mesh"
	// serviceLabel is the name of the label that holds the service name.
	serviceLabel = kumaMetaLabelPrefix + "service"
	// dataplaneLabel is the name of the label that holds the dataplane name.
	dataplaneLabel = kumaMetaLabelPrefix + "dataplane"
)

type KumaMadsAPIVersion string

const (
	KumaMadsV1 = KumaMadsAPIVersion("v1")
)

const (
	KumaMadsV1ResourceTypeURL = "type.googleapis.com/kuma.observability.v1.MonitoringAssignment"
	KumaMadsV1ResourceType    = "monitoringassignment"
)

type KumaSDConfig struct {
	xdsSDConfig `yaml:",inline"`
	// ClientName is sent to the xDS management API to identify this client
	ClientName string `yaml:"client_name"`
	// APIVersion is the MADS API version
	APIVersion KumaMadsAPIVersion `yaml:"api_version"`
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (c *KumaSDConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*c = DefaultKumaSDConfig
	type plain KumaSDConfig
	err := unmarshal((*plain)(c))
	if err != nil {
		return err
	}

	// Validate protocol
	switch c.ProtocolVersion {
	case ProtocolV3:
		break
	default:
		return fmt.Errorf("kuma SD only supports xDS v3: %s", c.ProtocolVersion)
	}

	// Validate apiVersion
	switch c.APIVersion {
	case KumaMadsV1:
		break
	default:
		return fmt.Errorf("kuma SD only supports MADS v1: %s", c.APIVersion)
	}

	if len(c.ClientName) == 0 {
		return errors.Errorf("kuma SD clientName must not be empty: %s", c.ClientName)
	}

	if len(c.Server) == 0 {
		return errors.Errorf("kuma SD server must not be empty: %s", c.Server)
	}
	parsedURL, err := url.Parse(c.Server)
	if err != nil {
		return err
	}

	if len(parsedURL.Scheme) == 0 || len(parsedURL.Host) == 0 {
		return errors.Errorf("kuma SD server must not be empty and have a scheme: %s", c.Server)
	}

	if c.HTTP == nil {
		return errors.Errorf("kuma SD server must not be empty: %s", c.Server)
	}

	if err := c.HTTP.Validate(); err != nil {
		return err
	}

	return nil
}

func (c *KumaSDConfig) Name() string {
	return "kuma"
}

func (c *KumaSDConfig) NewDiscoverer(opts discovery.DiscovererOptions) (discovery.Discoverer, error) {
	return NewKumaHTTPDiscovery(c, opts.Logger)
}

func convertKumaV1MonitoringAssignment(assignment *MonitoringAssignment) *targetgroup.Group {
	commonLabels := convertKumaLabels(assignment.Labels)

	commonLabels[meshLabel] = model.LabelValue(assignment.Mesh)
	commonLabels[serviceLabel] = model.LabelValue(assignment.Service)

	var targetLabelSets []model.LabelSet

	for _, target := range assignment.Targets {
		targetLabels := convertKumaLabels(target.Labels)

		targetLabels[dataplaneLabel] = model.LabelValue(target.Name)
		targetLabels[model.InstanceLabel] = model.LabelValue(target.Name)
		targetLabels[model.AddressLabel] = model.LabelValue(target.Address)
		targetLabels[model.SchemeLabel] = model.LabelValue(target.Scheme)
		targetLabels[model.MetricsPathLabel] = model.LabelValue(target.MetricsPath)

		targetLabelSets = append(targetLabelSets, targetLabels)
	}

	return &targetgroup.Group{
		Labels:  commonLabels,
		Targets: targetLabelSets,
	}
}

func convertKumaLabels(labels map[string]string) model.LabelSet {
	labelSet := model.LabelSet{}
	for key, value := range labels {
		name := kumaMetaLabelPrefix + strutil.SanitizeLabelName(key)
		labelSet[model.LabelName(name)] = model.LabelValue(value)
	}
	return labelSet
}

// kumaMadsV1ResourceParser is an xds.resourceParser
func kumaMadsV1ResourceParser(resources []*any.Any, typeURL string) ([]*targetgroup.Group, error) {
	if typeURL != KumaMadsV1ResourceTypeURL {
		return nil, errors.Errorf("recieved invalid typeURL for Kuma MADS v1 Resource: %s", typeURL)
	}

	var groups []*targetgroup.Group

	for _, resource := range resources {
		assignment := &MonitoringAssignment{}

		if err := json.Unmarshal(resource.Value, assignment); err != nil {
			return nil, err
		}

		groups = append(groups, convertKumaV1MonitoringAssignment(assignment))
	}

	return groups, nil
}

func NewKumaHTTPDiscovery(conf *KumaSDConfig, logger log.Logger) (discovery.Discoverer, error) {
	clientConfig := &HTTPResourceClientConfig{
		HTTPClientConfig: conf.HTTP.HTTPClientConfig,
		ResourceType:     KumaMadsV1ResourceType,
		ResourceTypeURL:  KumaMadsV1ResourceTypeURL,
		Server:           conf.Server,
		ClientID:         conf.ClientName,
	}

	client, err := NewHTTPResourceClient(clientConfig, conf.ProtocolVersion)
	if err != nil {
		return nil, err
	}

	d := &fetchDiscovery{
		client:         client,
		logger:         logger,
		source:         "kuma",
		parseResources: kumaMadsV1ResourceParser,
	}

	d.Discovery = refresh.NewDiscovery(
		logger,
		"kuma",
		time.Duration(conf.HTTP.RefreshInterval),
		d.refresh)

	return d, nil
}
