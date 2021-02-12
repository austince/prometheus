module github.com/prometheus/prometheus/discovery/xds

go 1.14

require (
	github.com/envoyproxy/go-control-plane v0.9.8
	github.com/envoyproxy/protoc-gen-validate v0.4.1 // indirect
	github.com/go-kit/kit v0.10.0
	github.com/prometheus/common v0.15.0
	github.com/prometheus/prometheus v0.0.0-00010101000000-000000000000
	golang.org/x/sys v0.0.0-20210112080510-489259a85091 // indirect
	google.golang.org/grpc v1.34.0
)

replace github.com/prometheus/prometheus => ../../
