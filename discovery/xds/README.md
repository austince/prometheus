# XDS Service Discovery

[xds-protocol]

https://github.com/gojek/consul-envoy-xds

Both HTTP and gRPC endpoints are available, though gRPC is preferred for scalability. Could feasibly support
both based on configuration. HTTP could use the `refresh` SD method, and gRPC could use streams.

Only xDS v3 is supported.

Need to fetch all [listeners](https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/listener/v3/listener.proto#envoy-v3-api-msg-config-listener-v3-listener).


[xds-protocol]: https://www.envoyproxy.io/docs/envoy/latest/api-docs/xds_protocol
