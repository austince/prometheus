module github.com/prometheus/prometheus/discovery/xds/api

go 1.14

require (
	//github.com/envoyproxy/go-control-plane v0.9.9-0.20210107210257-6e771204fae6
	//github.com/envoyproxy/protoc-gen-validate v0.4.1
	//github.com/go-kit/kit v0.10.0
	//github.com/golang/protobuf v1.4.3
	//github.com/prometheus/common v0.15.0
	//github.com/prometheus/prometheus v2.5.0+incompatible
	//google.golang.org/genproto v0.0.0-20210207032614-bba0dbe2a9ea
	//google.golang.org/grpc v1.34.0
	//google.golang.org/protobuf v1.25.0

	github.com/envoyproxy/go-control-plane v0.9.8
	github.com/envoyproxy/protoc-gen-validate v0.4.1
	github.com/golang/protobuf v1.4.3
	github.com/google/go-cmp v0.5.4 // indirect
	golang.org/x/net v0.0.0-20201224014010-6772e930b67b // indirect
	golang.org/x/sys v0.0.0-20210112080510-489259a85091 // indirect
	golang.org/x/text v0.3.4 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/genproto v0.0.0-20201201144952-b05cb90ed32e
	google.golang.org/grpc v1.34.0
	google.golang.org/protobuf v1.25.0
// When running `make generate` in this folder, one can get into errors of missing proto dependecies
// To solve the issue, uncomment the section below and run `go mod download`
//github.com/cncf/udpa v0.0.1
//github.com/envoyproxy/data-plane-api v0.0.0-20210211160942-18b54850c9b7
//github.com/googleapis/googleapis v0.0.0-20210210233221-66e0de02e649
)
