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
	v3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/discovery/refresh"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"google.golang.org/protobuf/types/known/anypb"
)

const (
	// metaLabelPrefix is the meta prefix used for all xDS meta labels.
	// in this discovery.
	metaLabelPrefix = model.MetaLabelPrefix + "xds_"
	// serverLabel is the label to notate the xDS Management Server
	// the targets were scraped from
	serverLabel = metaLabelPrefix + "server"
	// protocolVersionLabel is the label to note the xDS ProtocolVersion
	protocolVersionLabel = metaLabelPrefix + "protocol_version"
	// clientIDLabel is the name of the label that holds the client ID for HTTP-based mechanisms.
	clientIDLabel = metaLabelPrefix + "client_id"
)

// ProtocolVersion is the xDS protocol version
type ProtocolVersion string

const (
	ProtocolV3 = ProtocolVersion("v3")
)

type HTTPConfig struct {
	config.HTTPClientConfig `yaml:",inline"`
	RefreshInterval         model.Duration `yaml:"refresh_interval,omitempty"`
}

// SDConfig is a base config for xDS-based SD mechanisms
type SDConfig struct {
	HTTP            HTTPConfig      `yaml:",inline"`
	Server          string          `yaml:"server,omitempty"`
	ProtocolVersion ProtocolVersion `yaml:"protocol_version"`
	// ClientID is sent to the xDS management API to identify this client
	ClientID string `yaml:"client_id"`
}

func init() {
	// Register top-level SD Configs
	discovery.RegisterConfig(&KumaSDConfig{})
	// Register protobuf types that need to be marshalled/ unmarshalled
	// core xDS
	_ = protoTypes.RegisterMessage((&v3.DiscoveryRequest{}).ProtoReflect().Type())
	_ = protoTypes.RegisterMessage((&v3.DiscoveryResponse{}).ProtoReflect().Type())
	// implementations
	_ = protoTypes.RegisterMessage((&MonitoringAssignment{}).ProtoReflect().Type())
}

var protoTypes = new(protoregistry.Types)

func protoUnmarshalOptions() proto.UnmarshalOptions {
	return proto.UnmarshalOptions{
		DiscardUnknown: true,       // only want known fields
		Merge:          true,       // always using new messages
		Resolver:       protoTypes, // only want known types
	}
}

func protoJSONUnmarshalOptions() protojson.UnmarshalOptions {
	return protojson.UnmarshalOptions{
		DiscardUnknown: true,       // only want known fields
		Resolver:       protoTypes, // only want known types
	}
}

func protoJSONMarshalOptions() protojson.MarshalOptions {
	return protojson.MarshalOptions{
		UseProtoNames: true,
		Resolver:      protoTypes, // only want known types
	}
}

type resourceParser func(resources []*anypb.Any, typeUrl string) ([]*targetgroup.Group, error)

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

		group.Labels[clientIDLabel] = model.LabelValue(d.client.ID())
		group.Labels[serverLabel] = model.LabelValue(d.client.Server())
		group.Labels[protocolVersionLabel] = model.LabelValue(d.client.ProtocolVersion())
	}

	level.Debug(d.logger).Log("msg", "updated to version", "version", response.VersionInfo, "groups", len(parsedGroups))

	return parsedGroups, nil
}
