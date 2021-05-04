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
	"fmt"
	"github.com/go-kit/kit/log"
	"github.com/prometheus/common/config"
	"net/url"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/discovery/refresh"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/prometheus/prometheus/util/strutil"
	"google.golang.org/protobuf/types/known/anypb"
)

var (
	// DefaultKumaSDConfig is the default Kuma MADS SD configuration.
	DefaultKumaSDConfig = KumaSDConfig{
		SDConfig: SDConfig{
			HTTP: HTTPConfig{
				HTTPClientConfig: config.DefaultHTTPClientConfig,
				RefreshInterval:  model.Duration(30 * time.Second),
			},
			ProtocolVersion: ProtocolV3,
		},
		APIVersion: KumaMadsV1,
	}
)

const (
	// kumaMetaLabelPrefix is the meta prefix used for all kuma meta labels.
	// in this discovery.
	kumaMetaLabelPrefix = model.MetaLabelPrefix + "kuma_"

	// kumaMeshLabel is the name of the label that holds the mesh name.
	kumaMeshLabel = kumaMetaLabelPrefix + "mesh"
	// kumaServiceLabel is the name of the label that holds the service name.
	kumaServiceLabel = kumaMetaLabelPrefix + "service"
	// kumaDataplaneLabel is the name of the label that holds the dataplane name.
	kumaDataplaneLabel = kumaMetaLabelPrefix + "dataplane"
	// kumaMadsAPIVersionLabel is the name of the label that holds the MADS API version.
	kumaMadsAPIVersionLabel = kumaMetaLabelPrefix + "api_version"
	// kumaUserLabelPrefix is the name of the label that namespaces all user-defined labels.
	kumaUserLabelPrefix = kumaMetaLabelPrefix + "label_"
)

type KumaMadsAPIVersion string

const (
	KumaMadsV1 = KumaMadsAPIVersion("v1")
)

const (
	KumaMadsV1ResourceTypeURL = "type.googleapis.com/kuma.observability.v1.MonitoringAssignment"
	KumaMadsV1ResourceType    = "monitoringassignments"
)

type KumaSDConfig struct {
	SDConfig `yaml:",inline"`
	// APIVersion is the MADS API version
	APIVersion KumaMadsAPIVersion `yaml:"api_version"`
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (c *KumaSDConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*c = DefaultKumaSDConfig
	type plainKumaConf KumaSDConfig
	err := unmarshal((*plainKumaConf)(c))
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

	if len(c.ClientID) == 0 {
		return errors.Errorf("kuma SD clientName must not be empty: %s", c.ClientID)
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

	if err := c.HTTP.Validate(); err != nil {
		return err
	}

	return nil
}

func (c *KumaSDConfig) Name() string {
	return "kuma"
}

// SetDirectory joins any relative file paths with dir.
func (c *KumaSDConfig) SetDirectory(dir string) {
	c.HTTP.HTTPClientConfig.SetDirectory(dir)
}

func (c *KumaSDConfig) NewDiscoverer(opts discovery.DiscovererOptions) (discovery.Discoverer, error) {
	return NewKumaHTTPDiscovery(c, opts.Logger)
}

func convertKumaV1MonitoringAssignment(assignment *MonitoringAssignment) *targetgroup.Group {
	commonLabels := convertKumaUserLabels(assignment.Labels)

	commonLabels[kumaMeshLabel] = model.LabelValue(assignment.Mesh)
	commonLabels[kumaServiceLabel] = model.LabelValue(assignment.Service)
	commonLabels[kumaMadsAPIVersionLabel] = model.LabelValue(KumaMadsV1)

	var targetLabelSets []model.LabelSet

	for _, target := range assignment.Targets {
		targetLabels := convertKumaUserLabels(target.Labels)

		targetLabels[kumaDataplaneLabel] = model.LabelValue(target.Name)
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

func convertKumaUserLabels(labels map[string]string) model.LabelSet {
	labelSet := model.LabelSet{}
	for key, value := range labels {
		name := kumaUserLabelPrefix + strutil.SanitizeLabelName(key)
		labelSet[model.LabelName(name)] = model.LabelValue(value)
	}
	return labelSet
}

// kumaMadsV1ResourceParser is an xds.resourceParser
func kumaMadsV1ResourceParser(resources []*anypb.Any, typeURL string) ([]*targetgroup.Group, error) {
	if typeURL != KumaMadsV1ResourceTypeURL {
		return nil, errors.Errorf("recieved invalid typeURL for Kuma MADS v1 Resource: %s", typeURL)
	}

	var groups []*targetgroup.Group

	for _, resource := range resources {
		assignment := &MonitoringAssignment{}

		if err := anypb.UnmarshalTo(resource, assignment, protoUnmarshalOptions()); err != nil {
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
		ClientID:         conf.ClientID,
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
