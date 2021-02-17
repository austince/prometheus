module github.com/prometheus/prometheus/discovery/xds/poc_server

go 1.14

replace github.com/prometheus/prometheus/discovery/xds/api => ../api

require (
	github.com/envoyproxy/go-control-plane v0.9.8
	github.com/golang/protobuf v1.4.3
	github.com/prometheus/prometheus/discovery/xds/api v0.0.0-00010101000000-000000000000
)
