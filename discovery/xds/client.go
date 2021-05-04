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
	"bytes"
	"context"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"

	envoy_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	v3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"

	"github.com/prometheus/common/config"
	"github.com/prometheus/common/version"
)

var userAgent = fmt.Sprintf("Prometheus/%s", version.Version)

// ResourceClient exposes the xDS protocol for a single resource type
// It handles caching w/ version and nonce mgmt.
// Only the State of the World protocol variant is supported at this time.
// See https://www.envoyproxy.io/docs/envoy/latest/api-docs/xds_protocol#rest-json-polling-subscriptions
type ResourceClient interface {
	// ResourceTypeUrl is the type URL of the resource
	ResourceTypeURL() string

	// Server is the xDS Management server
	Server() string

	// ProtocolVersion is the xDS protocol in use
	ProtocolVersion() ProtocolVersion

	// Fetch requests the latest view of the entire resource state
	// If no updates have been made since the last request, the response will be nil
	Fetch(ctx context.Context) (*v3.DiscoveryResponse, error)

	// ID returns the ID of the client that is sent to the xDS server
	ID() string
}

type HTTPResourceClient struct {
	client          *http.Client
	config          *HTTPResourceClientConfig
	protocolVersion ProtocolVersion
	server          string
	// endpoint is the fully-constructed xDS HTTP endpoint
	endpoint string
	// Caching
	latestVersion string
	latestNonce   string
}

type HTTPResourceClientConfig struct {
	// HTTP config
	config.HTTPClientConfig
	// General xDS config

	// Type is the xds type, e.g., clusters
	// which is used in the discovery POST request
	ResourceType string
	// ResourceTypeURL is the Google type url for the resource, e.g., type.googleapis.com/envoy.api.v2.Cluster
	ResourceTypeURL string
	// Server is the xDS management server
	Server string
	// ClientID is used to identify the client with the management server
	ClientID string
}

func NewHTTPResourceClient(conf *HTTPResourceClientConfig, protocolVersion ProtocolVersion) (*HTTPResourceClient, error) {
	rt, err := config.NewRoundTripperFromConfig(conf.HTTPClientConfig, "xds_http_client", config.WithHTTP2Disabled())
	if err != nil {
		return nil, err
	}

	if len(conf.Server) == 0 {
		return nil, errors.New("xds_sd: empty or null xDS server")
	}

	serverURL, err := url.Parse(conf.Server)
	if err != nil {
		return nil, err
	}

	endpoint, err := makeXDSResourceHTTPEndpoint(protocolVersion, serverURL, conf.ResourceType)
	if err != nil {
		return nil, err
	}

	return &HTTPResourceClient{
		client:          &http.Client{Transport: rt},
		config:          conf,
		protocolVersion: protocolVersion,
		server:          conf.Server,
		endpoint:        endpoint,
		latestVersion:   "",
		latestNonce:     "",
	}, nil
}

func makeXDSResourceHTTPEndpoint(protocolVersion ProtocolVersion, serverURL *url.URL, resourceType string) (string, error) {
	switch protocolVersion {
	case ProtocolV3:
		break
	default:
		return "", errors.Errorf("xds_http_client: unsupported xDS protocol version: %s", protocolVersion)
	}

	if serverURL == nil {
		return "", errors.New("xds_http_client: empty xDS server URL")
	}

	if len(serverURL.Scheme) == 0 || len(serverURL.Host) == 0 {
		return "", errors.New("xds_http_client: invalid xDS server URL")
	}

	if serverURL.Scheme != "http" && serverURL.Scheme != "https" {
		return "", errors.New("xds_http_client: invalid xDS server URL protocol. must be either 'http' or 'https'")
	}

	serverURL.Path = path.Join(serverURL.Path, string(protocolVersion), fmt.Sprintf("discovery:%s", resourceType))

	return serverURL.String(), nil
}

func (rc *HTTPResourceClient) Server() string {
	return rc.config.Server
}

func (rc *HTTPResourceClient) ResourceTypeURL() string {
	return rc.config.ResourceTypeURL
}

func (rc *HTTPResourceClient) ProtocolVersion() ProtocolVersion {
	return rc.protocolVersion
}

func (rc *HTTPResourceClient) ID() string {
	return rc.config.ClientID
}

// Fetch requests the latest state of the resources from the xDS server
func (rc *HTTPResourceClient) Fetch(ctx context.Context) (*v3.DiscoveryResponse, error) {
	discoveryReq := &v3.DiscoveryRequest{
		VersionInfo:   rc.latestVersion,
		ResponseNonce: rc.latestNonce,
		TypeUrl:       rc.ResourceTypeURL(),
		ResourceNames: []string{},
		Node: &envoy_core.Node{
			Id: rc.config.ClientID,
		},
	}

	reqBody, err := protoJSONMarshalOptions().Marshal(discoveryReq)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest("POST", rc.endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	request = request.WithContext(ctx)

	request.Header.Add("User-Agent", userAgent)
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Accept", "application/json")

	resp, err := rc.client.Do(request)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == 304 {
		// Empty response, already have the latest
		return nil, nil
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("non 200 status '%d' response during xDS fetch", resp.StatusCode)
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	defer func() {
		_, _ = io.Copy(ioutil.Discard, resp.Body)
		_ = resp.Body.Close()
	}()
	if err != nil {
		return nil, err
	}

	discoveryRes := &v3.DiscoveryResponse{}
	if err = protoJSONUnmarshalOptions().Unmarshal(respBody, discoveryRes); err != nil {
		return nil, err
	}

	// cache latest nonce + version info
	rc.latestNonce = discoveryRes.Nonce
	rc.latestVersion = discoveryRes.VersionInfo

	return discoveryRes, nil
}
