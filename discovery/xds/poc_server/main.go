package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	v3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/prometheus/prometheus/discovery/xds/api"
	"github.com/prometheus/prometheus/discovery/xds/api/v1alpha1"
)

// Just a POC
// Servers (in go) should be implemented with github.com/envoyproxy/go-control-plane/pkg/server

func main() {
	assignment := &v1alpha1.MonitoringAssignment{
		Source: "test-svc",
		Targets: []*v1alpha1.MonitoringAssignment_Target{
			{
				Address: "127.0.0.1:6767",
				Instance: "testerooni-0",
				MetricsPath: "/metrics",

				Labels: map[string]string{
					"commit_hash": "620506a88",
				},
			},
			{
				Address: "127.0.0.1:6767",
				Instance: "testerooni-1",

				Labels: map[string]string{
					"commit_hash": "620506a88",
				},
			},
		},
		Labels: map[string]string{
			"mesh": "default",
		},
	}

	serializedAssignment, err := json.Marshal(assignment)
	if err != nil {
		panic(err)
	}

	resources := []*any.Any{
		{
			TypeUrl: api.MonitoringAssignmentTypeUrl,
			Value:   serializedAssignment,
		},
	}

	discoveryRes := v3.DiscoveryResponse{
		VersionInfo: "1",
		Resources:   resources,
		ControlPlane: nil,
		TypeUrl:     api.MonitoringAssignmentTypeUrl,
		Nonce:       "",
	}

	serializedDiscoveryRes, err := json.Marshal(discoveryRes)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Sample assignment: \n{%v}\n", assignment)

	// mock assignments
	http.HandleFunc("/v3/discovery:monitoringassignment", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("%v: Handling discovery\n", time.Now().Format(time.Kitchen))

		decoder := json.NewDecoder(r.Body)
		var req v3.DiscoveryRequest
		if err := decoder.Decode(&req); err != nil {
			w.WriteHeader(400)
			w.Write([]byte(fmt.Sprintf(`{"error": "Cannot decode body", message: "%s"}`, err.Error())))
			fmt.Printf("Error decoding body %v\n", err)
			return
		}
		fmt.Printf("Discovery request: %v\n", req)

		w.WriteHeader(200)
		if _, err = w.Write(serializedDiscoveryRes); err != nil {
			fmt.Printf("Error writing bytes %s\n", err.Error())
		}
	})

	// mock metrics
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Handling metrics on /metrics path")
		// TODO
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Handling metrics on root path")
		// TODO
	})

	fmt.Printf("POC xDS server listening on port 6767\n")
	log.Fatal(http.ListenAndServe(":6767", nil))
}
