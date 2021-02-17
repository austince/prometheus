package xds

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	v3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/pkg/errors"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery/xds/api"
	"github.com/prometheus/prometheus/discovery/xds/api/v1alpha1"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/prometheus/common/config"
	"github.com/prometheus/common/version"
	"github.com/prometheus/prometheus/discovery/refresh"
	"github.com/prometheus/prometheus/discovery/targetgroup"
)

var userAgent = fmt.Sprintf("Prometheus/%s", version.Version)
const v3MonitoringDiscoveryPath = "/v3/discovery:monitoring"

type HTTPDiscovery struct {
	server string
	*refresh.Discovery
	log log.Logger
	client *http.Client
	conf *SDConfig
}

func newHttpDiscovery(conf *SDConfig, logger log.Logger) (*HTTPDiscovery, error) {
	rt, err := config.NewRoundTripperFromConfig(conf.Http.HTTPClientConfig, "xds_sd", false, false)
	if err != nil {
		return nil, err
	}

	d := &HTTPDiscovery{
		client: &http.Client{Transport: rt},
		server: conf.Server,
		log: logger,
		conf: conf,
	}
	d.Discovery = refresh.NewDiscovery(
		logger,
		"xds",
		time.Duration(conf.Http.RefreshInterval),
		d.refresh,
	)
	return d, nil
}

// TODO: implement/ use an actual xDS client
func (d *HTTPDiscovery) fetchDiscovery(ctx context.Context) (*v3.DiscoveryResponse, error) {
	url := fmt.Sprintf("%s%s", d.server, v3MonitoringDiscoveryPath)

	discoveryReq := &v3.DiscoveryRequest{
		VersionInfo: "v1beta1", // probably not correct
		Node: nil, // TODO
		ResourceNames: []string{},
		TypeUrl: api.MonitoringAssignmentTypeUrl,
		ResponseNonce: "",
		ErrorDetail: nil,
	}

	reqBody, err := json.Marshal(discoveryReq)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	request = request.WithContext(ctx)
	request.Header.Add("User-Agent", userAgent)

	resp, err := d.client.Do(request)
	if err != nil {
		return nil, err
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != 200 {
		return nil, errors.Errorf("non 200 status '%d' response during xDS service discovery", resp.StatusCode)
	}

	d.log.Log("Response", string(respBody))

	discoveryRes := &v3.DiscoveryResponse{}
	if err = json.Unmarshal(respBody, discoveryRes); err != nil {
		return nil, err
	}

	return discoveryRes, nil
}

func (d *HTTPDiscovery) refresh(ctx context.Context) ([]*targetgroup.Group, error) {
	discoveryRes, err := d.fetchDiscovery(ctx)
	if err != nil {
		return nil, err
	}

	var assignments []*v1alpha1.MonitoringAssignment
	for _, res := range discoveryRes.Resources {

		assignment := &v1alpha1.MonitoringAssignment{}
		if err := json.Unmarshal(res.Value, assignment); err != nil {
			d.log.Log("discovery fetch error", err)
			return nil, err
		}

		assignments = append(assignments, assignment)
	}

	// convert assignments to target groups
	groups := []*targetgroup.Group{}

	for _, assignment := range assignments {
		var targets []model.LabelSet

		for _, assignmentTarget := range assignment.Targets {
			targetLabels := model.LabelSet{}

			// map labels for the single assignmentTarget
			for name, val := range assignmentTarget.Labels {
				targetLabels[prefixedLabel(name)] = lv(val)
			}

			targetLabels[model.AddressLabel] = lv(assignmentTarget.Address)
			targetLabels[model.InstanceLabel] = lv(assignmentTarget.Instance)

			if len(assignmentTarget.MetricsPath) > 0 {
				targetLabels[model.MetricsPathLabel] = lv(assignmentTarget.MetricsPath)
			}

			targets = append(targets, targetLabels)
		}

		groupLabels := model.LabelSet{}
		for name, val := range assignment.Labels {
			groupLabels[prefixedLabel(name)] = lv(val)
		}

		groupLabels[nameLabel] = lv(assignment.Name)
		groupLabels[serverLabel] = lv(d.conf.Server)
		groupLabels[apiVersion] = lv(string(d.conf.ApiVersion))
		groupLabels[protocolVersion] = lv(string(d.conf.ProtocolVersion))

		tg := &targetgroup.Group{
			Source: source,
			Targets: targets,
			Labels: groupLabels,
		}

		groups = append(groups, tg)
	}



	return groups, nil
}

func prefixedLabel(s string) model.LabelName {
	return model.LabelName(metaLabelPrefix + s)
}

// lv is just a shorthand
func lv(s string) model.LabelValue {
	return model.LabelValue(s)
}
