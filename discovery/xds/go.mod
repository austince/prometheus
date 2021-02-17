module github.com/prometheus/prometheus/discovery/xds

go 1.14

require (
	github.com/cncf/udpa v0.0.2-0.20201211205326-cc1b757b3edd // indirect
	github.com/envoyproxy/data-plane-api v0.0.0-20210105195927-01fb099f5a86 // indirect
	github.com/envoyproxy/go-control-plane v0.9.8
	github.com/envoyproxy/protoc-gen-validate v0.4.1 // indirect
	github.com/go-kit/kit v0.10.0
	github.com/googleapis/googleapis v0.0.0-20210217154535-8b0cc14345ff // indirect
	github.com/pkg/errors v0.9.1
	github.com/prometheus/common v0.15.0
	github.com/prometheus/prometheus v0.0.0-00010101000000-000000000000
	google.golang.org/grpc v1.34.0
)

replace github.com/prometheus/prometheus => ../../
