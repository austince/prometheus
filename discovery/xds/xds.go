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
	"context"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/pkg/errors"
	"github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/discovery/refresh"
	"github.com/prometheus/prometheus/discovery/targetgroup"
)

const (
	// metaLabelPrefix is the meta prefix used for all xDS meta labels.
	// in this discovery.
	metaLabelPrefix = model.MetaLabelPrefix + "xds_"
	// serverLabel is the label to notate the xDS Management Server
	// the targets were scraped from
	serverLabel          = metaLabelPrefix + "server"
	protocolVersionLabel = metaLabelPrefix + "protocol_version"
)

// The xDS protocol version
type ProtocolVersion string

const (
	ProtocolV3 = ProtocolVersion("v3")
)

type HTTPConfig struct {
	config.HTTPClientConfig `yaml:",inline"`
	RefreshInterval         model.Duration `yaml:"refresh_interval,omitempty"`
}

type xdsSDConfig struct {
	Server          string          `yaml:"server,omitempty"`
	HTTP            *HTTPConfig     `yaml:"http,omitempty"`
	ProtocolVersion ProtocolVersion `yaml:"protocol_version"`
}

func init() {
	discovery.RegisterConfig(&KumaSDConfig{})
}

type resourceParser func(resources []*any.Any, typeUrl string) ([]*targetgroup.Group, error)

type fetchDiscovery struct {
	*refresh.Discovery

	client ResourceClient
	source string

	parseResources resourceParser
	logger         log.Logger
}

func (d *fetchDiscovery) refresh(ctx context.Context) ([]*targetgroup.Group, error) {
	response, err := d.client.Fetch(ctx)
	if err != nil {
		return nil, err
	}

	if response == nil {
		return nil, errors.Wrapf(refresh.ErrSkipUpdate, "resource version up to date: %s", d.client.ResourceTypeURL())
	}

	level.Debug(d.logger).Log("msg", "fetched response", "response", response)

	parsedGroups, err := d.parseResources(response.Resources, response.TypeUrl)
	if err != nil {
		return nil, err
	}

	for _, group := range parsedGroups {
		group.Source = d.source

		if group.Labels == nil {
			group.Labels = model.LabelSet{}
		}

		group.Labels[serverLabel] = model.LabelValue(d.client.Server())
		group.Labels[protocolVersionLabel] = model.LabelValue(d.client.ProtocolVersion())
	}

	level.Debug(d.logger).Log("msg", "updated to version", "version", response.VersionInfo, "groups", parsedGroups)
	return parsedGroups, nil
}
