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
	"errors"
	v3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/prometheus/common/config"
	"github.com/stretchr/testify/require"
	"net/url"
	"testing"
)

var (
	httpResourceConf = &HTTPResourceClientConfig{
		HTTPClientConfig: config.HTTPClientConfig{
			TLSConfig: config.TLSConfig{InsecureSkipVerify: true},
		},
		ResourceType:    "monitoring",
		ResourceTypeURL: "type.googleapis.com/some.api.v1.Monitoring",
		Server:          "http://localhost",
		ClientID:        "testing",
	}
)

func urlMustParse(str string) *url.URL {
	parsed, err := url.Parse(str)

	if err != nil {
		panic(err)
	}

	return parsed
}

func TestMakeXDSResourceHttpEndpointEmptyServerURLScheme(t *testing.T) {
	endpointURL, err := makeXDSResourceHTTPEndpoint(ProtocolV3, urlMustParse("127.0.0.1"), "monitoring")

	require.Empty(t, endpointURL)
	require.Error(t, err)
	require.Equal(t, err.Error(), "xds_http_client: invalid xDS server URL")
}

func TestMakeXDSResourceHttpEndpointEmptyServerURLHost(t *testing.T) {
	endpointURL, err := makeXDSResourceHTTPEndpoint(ProtocolV3, urlMustParse("grpc://127.0.0.1"), "monitoring")

	require.Empty(t, endpointURL)
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "must be either 'http' or 'https'")
}

func TestMakeXDSResourceHttpEndpoint(t *testing.T) {
	endpointURL, err := makeXDSResourceHTTPEndpoint(ProtocolV3, urlMustParse("http://127.0.0.1:5000"), "monitoring")

	require.NoError(t, err)
	require.Equal(t, endpointURL, "http://127.0.0.1:5000/v3/discovery:monitoring")
}

func createTestHTTPResourceClient(t *testing.T, conf *HTTPResourceClientConfig, protocolVersion ProtocolVersion, responder discoveryResponder) (*HTTPResourceClient, func()) {
	s := createTestHTTPServer(t, func(request *v3.DiscoveryRequest) (*v3.DiscoveryResponse, error) {
		require.Equal(t, conf.ResourceTypeURL, request.TypeUrl)
		require.Equal(t, conf.ClientID, request.Node.Id)
		return responder(request)
	})

	conf.Server = s.URL
	client, err := NewHTTPResourceClient(conf, protocolVersion)
	require.NoError(t, err)

	return client, s.Close
}

func TestHTTPResourceClientFetchEmptyResponse(t *testing.T) {
	client, cleanup := createTestHTTPResourceClient(t, httpResourceConf, ProtocolV3, func(request *v3.DiscoveryRequest) (*v3.DiscoveryResponse, error) {
		return nil, nil
	})
	defer cleanup()

	res, err := client.Fetch(context.Background())
	require.Nil(t, res)
	require.NoError(t, err)
}

func TestHTTPResourceClientFetchFullResponse(t *testing.T) {
	client, cleanup := createTestHTTPResourceClient(t, httpResourceConf, ProtocolV3, func(request *v3.DiscoveryRequest) (*v3.DiscoveryResponse, error) {
		if request.VersionInfo == "1" {
			return nil, nil
		}

		return &v3.DiscoveryResponse{
			TypeUrl:     request.TypeUrl,
			VersionInfo: "1",
			Nonce:       "abc",
			Resources: []*any.Any{
				{
					TypeUrl: request.TypeUrl,
					Value:   []byte("some-value"),
				},
			},
		}, nil
	})
	defer cleanup()

	res, err := client.Fetch(context.Background())
	require.NoError(t, err)
	require.NotNil(t, res)

	require.Equal(t, client.ResourceTypeURL(), res.TypeUrl)
	require.Len(t, res.Resources, 1)
	// should cache the nonce and version info
	require.Equal(t, "abc", client.latestNonce)
	require.Equal(t, "1", client.latestVersion)

	// should send cached version info in subsequent requests
	res, err = client.Fetch(context.Background())
	require.Nil(t, res)
	require.NoError(t, err)
}

func TestHTTPResourceClientServerError(t *testing.T) {
	client, cleanup := createTestHTTPResourceClient(t, httpResourceConf, ProtocolV3, func(request *v3.DiscoveryRequest) (*v3.DiscoveryResponse, error) {
		return nil, errors.New("server error")
	})
	defer cleanup()

	res, err := client.Fetch(context.Background())
	require.Nil(t, res)
	require.Error(t, err)
}
