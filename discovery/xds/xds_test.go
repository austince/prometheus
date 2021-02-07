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
	"encoding/json"
	"errors"
	"github.com/prometheus/prometheus/discovery/refresh"
	"go.uber.org/goleak"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	v3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/stretchr/testify/require"
)

var (
	conf = xdsSDConfig{
		Server:          "http://127.0.0.1",
		ProtocolVersion: ProtocolV3,
		HTTP: &HTTPConfig{
			RefreshInterval: mustParseDuration("10s"),
		},
	}

	kumaConf = KumaSDConfig{
		xdsSDConfig: conf,
		ClientName:  "test_client",
		APIVersion:  KumaMadsV1,
	}
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

type discoveryResponder func(request *v3.DiscoveryRequest) (*v3.DiscoveryResponse, error)

func createTestHTTPServer(t *testing.T, responder discoveryResponder) *httptest.Server {
	return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// validate req MIME types
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))
		require.Equal(t, "application/json", r.Header.Get("Accept"))

		body, err := ioutil.ReadAll(r.Body)
		defer func() {
			_, _ = io.Copy(ioutil.Discard, r.Body)
			_ = r.Body.Close()
		}()
		require.NotEmpty(t, body)
		require.NoError(t, err)

		// validate discovery request
		discoveryReq := &v3.DiscoveryRequest{}
		err = json.Unmarshal(body, discoveryReq)
		require.NoError(t, err)

		discoveryRes, err := responder(discoveryReq)
		if err != nil {
			w.WriteHeader(500)
			return
		}

		if discoveryRes == nil {
			w.WriteHeader(304)
		} else {
			w.WriteHeader(200)
			data, err := json.Marshal(discoveryRes)
			require.NoError(t, err)

			_, err = w.Write(data)
			require.NoError(t, err)
		}
	}))
}

func constantResourceParser(groups []*targetgroup.Group, err error) resourceParser {
	return func(resources []*any.Any, typeUrl string) ([]*targetgroup.Group, error) {
		return groups, err
	}
}

type devNullLogger byte

func (devNullLogger) Log(...interface{}) error {
	return nil
}

const nullLogger = devNullLogger(0)

type testResourceClient struct {
	resourceTypeURL string
	server          string
	protocolVersion ProtocolVersion
	fetch           func(ctx context.Context) (*v3.DiscoveryResponse, error)
}

func (rc testResourceClient) ResourceTypeURL() string {
	return rc.resourceTypeURL
}

func (rc testResourceClient) Server() string {
	return rc.server
}

func (rc testResourceClient) ProtocolVersion() ProtocolVersion {
	return rc.protocolVersion
}

func (rc testResourceClient) Fetch(ctx context.Context) (*v3.DiscoveryResponse, error) {
	return rc.fetch(ctx)
}

func mustParseDuration(s string) model.Duration {
	dur, err := model.ParseDuration(s)
	if err != nil {
		panic("could not parse duration(\"" + s + "\"): " + err.Error())
	}
	return dur
}

func TestPollingRefreshSkipUpdate(t *testing.T) {
	rc := &testResourceClient{
		fetch: func(ctx context.Context) (*v3.DiscoveryResponse, error) {
			return nil, nil
		},
	}
	pd := &fetchDiscovery{
		client: rc,
	}
	groups, err := pd.refresh(context.Background())
	require.Nil(t, groups)
	require.True(t, errors.Is(err, refresh.ErrSkipUpdate))
}

func TestPollingRefreshAttachesGroupMetadata(t *testing.T) {
	server := "http://198.161.2.0"
	source := "test"
	rc := &testResourceClient{
		server:          server,
		protocolVersion: ProtocolV3,
		fetch: func(ctx context.Context) (*v3.DiscoveryResponse, error) {
			return &v3.DiscoveryResponse{}, nil
		},
	}
	pd := &fetchDiscovery{
		source: source,
		client: rc,
		logger: nullLogger,
		parseResources: constantResourceParser([]*targetgroup.Group{
			{},
			{
				Source: "a-custom-source",
				Labels: model.LabelSet{
					serverLabel:          "a-server",
					protocolVersionLabel: "a-version",
					"__meta_xsd_a_label": "a-value",
				},
			},
		}, nil),
	}
	groups, err := pd.refresh(context.Background())
	require.NotNil(t, groups)
	require.Nil(t, err)

	require.Len(t, groups, 2)

	for _, group := range groups {
		require.Equal(t, source, group.Source)
		labels := group.Labels
		require.NotNil(t, labels)
		require.Equal(t, server, string(labels[serverLabel]))
		require.Equal(t, ProtocolV3, ProtocolVersion(labels[protocolVersionLabel]))
	}

	group2 := groups[1]
	require.Contains(t, group2.Labels, model.LabelName("__meta_xsd_a_label"))
	require.Equal(t, model.LabelValue("a-value"), group2.Labels["__meta_xsd_a_label"])
}
