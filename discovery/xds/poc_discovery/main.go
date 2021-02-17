package main

import (
	"context"
	"fmt"
	"github.com/prometheus/common/model"

	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/prometheus/prometheus/discovery/xds"
)

type logger struct {
}

func (l *logger) Log(keyvals ...interface{}) error {
	fmt.Println(keyvals...)
	return nil
}

func main() {
	dur, err := model.ParseDuration("10s")
	if err != nil {
		panic(err)
	}

	conf := &xds.SDConfig{
		Mode: xds.HTTPMode,
		Server: "http://localhost:6767",
		ProtocolVersion: "v3",
		ApiVersion: "v1alpha1",
		Http: &xds.HTTPConfig{
			RefreshInterval: dur,
		},
	}

	opts := discovery.DiscovererOptions{
		Logger: &logger{},
	}

	d, err := conf.NewDiscoverer(opts)
	if err != nil {
		fmt.Printf("err %v\n", err)
		return
	}

	c := make(chan []*targetgroup.Group)

	go d.Run(context.TODO(), c)

	defer close(c)

	for groups := range c {
		fmt.Printf("Got groups %v\n", groups)
		for _, g := range groups {
			fmt.Println()
			fmt.Printf("group %v\n", g)
			fmt.Printf("with labels %v\n", g.Labels)
			fmt.Printf("with targets %v\n", g.Targets)
			fmt.Printf("with source %v\n", g.Source)
		}
		fmt.Println()
	}
	fmt.Println()
}
