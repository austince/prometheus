# XDS Service Discovery

This is a Proof of Concept implementation of an xDS discovery mechanism, Envoyproxy's discovery protocol, in Prometheus.
It involves two main parts: the discovery scraping and the xDS API implementation. 

## Discovery Scraping

This is the most similar to other discovery mechanisms that exist in Prometheus today. 
As described in the [xDS Protocol][xds-protocol], both HTTP and gRPC are valid transports though they have different
semantics and capabilities. In short, HTTP is much simpler but is less efficient as more data needs to be sent with each
request/ response.

### Notes

* All scraped labels are prefixed with `__meta_xds_`
* "Deltas" (only fetching what's changed) is only possible with gRPC. More can be found on [gRPC streaming here][xds-grpc-streaming].

## xDS API Implementation

Envoy does not have a native concept of Metrics Targets, so there is no resource "builtin" Envoy's xDS that represents the
data necessary for Prometheus scraping ([all v3 services here][xds-services]). But the protocol is designed to be easily extended!
The [`v1alpha1.MonitoringAssignment`](api/v1alpha1/mads.proto) does just that, roughly based on
[`targetgroup.Group`](../targetgroup/targetgroup.go) with more explicit label fields for the necessary info.

Both HTTP and gRPC endpoints are available, though gRPC is preferred for scalability. We can feasibly support
both based on user configuration. HTTP uses the `refresh` SD util, and gRPC could use streams (when implemented).

Only xDS v3 is supported.

### Building the Protos

Building requires [`protoc`][protoc] to be installed. 

After that, run:

```shell
make install
make generate
```

This should compile the `.proto` files in [`api/v1alpha1`](api/v1alpha1) into `.go` implementations in the same directory.

## PoC Discovery and Server

There are two small programs to demonstrate the interaction between the discovery mechanism and the xDS server. The
program in [`poc_server`](poc_server) runs a mock xDS HTTP server for MonitoringAssignments on port `6767`.
The program in [`poc_discovery`](poc_discovery) runs the `http_discovery`, which periodically scrapes from the mock server
and prints the received target groups.

In one terminal:
`make run/server`

In another:
`make run/discovery`

For gRPC clients/ servers, a sample implementation of both can be found [here](https://github.com/kumahq/kuma/tree/d6ac4b483c5166d4b92b4d9f0e7e59b103d3afde/pkg/mads/).

## Wrapping Up

The big changes to Prometheus would then be:
* Added dependencies on Envoy and gRPC
* Added api to maintain, version, add to build process
  * This would likely be very stable
* Docs for implementing this type of xDS server
* ...I'm sure plenty more? 


[xds-protocol]: https://www.envoyproxy.io/docs/envoy/latest/api-docs/xds_protocol
[xds-grpc-streaming]: https://www.envoyproxy.io/docs/envoy/latest/api-docs/xds_protocol#streaming-grpc-subscriptions
[xds-services]: https://www.envoyproxy.io/docs/envoy/latest/api-v3/service/service
[protoc]: https://grpc.io/docs/protoc-installation/
