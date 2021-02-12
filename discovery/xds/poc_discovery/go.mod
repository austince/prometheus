module github.com/prometheus/prometheus/discovery/poc_server

go 1.14

replace (
	github.com/prometheus/prometheus => ../../../
	github.com/prometheus/prometheus/discovery/xds => ../
	github.com/prometheus/prometheus/discovery/xds/api => ../api
)

require (
	github.com/prometheus/prometheus/discovery/xds v0.0.0-00010101000000-000000000000
	github.com/prometheus/prometheus/discovery/xds/api v0.0.0-00010101000000-000000000000 // indirect
)
