# XDS Service Discovery

## Discovery Scraping

This is the most similar to other discovery mechanisms that exist in Prometheus today. 
As described in the [xDS Protocol][xds-protocol], both HTTP and gRPC are valid transports though they have different
semantics and capabilities. In short, HTTP is much simpler but is less efficient as more data needs to be sent with each
request/ response.

## xDS API Implementation

Only xDS v3 and HTTP are supported.

[xds-protocol]: https://www.envoyproxy.io/docs/envoy/latest/api-docs/xds_protocol
[xds-grpc-streaming]: https://www.envoyproxy.io/docs/envoy/latest/api-docs/xds_protocol#streaming-grpc-subscriptions
[xds-services]: https://www.envoyproxy.io/docs/envoy/latest/api-v3/service/service
[protoc]: https://grpc.io/docs/protoc-installation/
