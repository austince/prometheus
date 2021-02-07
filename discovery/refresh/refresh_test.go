// Copyright 2019 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package refresh

import (
	"context"
	"errors"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"testing"
	"time"

	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/prometheus/prometheus/discovery/targetgroup"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestRefresh(t *testing.T) {
	resetMetrics()
	tg1 := []*targetgroup.Group{
		{
			Source: "tg",
			Targets: []model.LabelSet{
				{
					model.LabelName("t1"): model.LabelValue("v1"),
				},
				{
					model.LabelName("t2"): model.LabelValue("v2"),
				},
			},
			Labels: model.LabelSet{
				model.LabelName("l1"): model.LabelValue("lv1"),
			},
		},
	}
	tg2 := []*targetgroup.Group{
		{
			Source: "tg",
		},
	}

	var i int
	refresh := func(ctx context.Context) ([]*targetgroup.Group, error) {
		i++
		switch i {
		case 1:
			return tg1, nil
		case 2:
			return tg2, nil
		}
		return nil, fmt.Errorf("some error")
	}
	interval := time.Millisecond
	d := NewDiscovery(nil, "test", interval, refresh)

	ch := make(chan []*targetgroup.Group)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go d.Run(ctx, ch)

	tg := <-ch
	require.Equal(t, tg1, tg)

	tg = <-ch
	require.Equal(t, tg2, tg)

	tick := time.NewTicker(2 * interval)
	defer tick.Stop()
	select {
	case <-ch:
		t.Fatal("Unexpected target group")
	case <-tick.C:
	}
}

func TestIncFailures(t *testing.T) {
	resetMetrics()
	tg1 := []*targetgroup.Group{
		{
			Source: "tg",
			Targets: []model.LabelSet{
				{
					model.LabelName("t1"): model.LabelValue("v1"),
				},
			},
			Labels: model.LabelSet{
				model.LabelName("l1"): model.LabelValue("lv1"),
			},
		},
	}
	tg2 := []*targetgroup.Group{
		{
			Source: "tg",
		},
	}

	var i int
	refresh := func(ctx context.Context) ([]*targetgroup.Group, error) {
		i++
		switch i {
		case 1:
			return tg1, nil
		case 2:
			return nil, errors.New("some error")
		}
		return tg2, nil
	}
	interval := time.Millisecond
	d := NewDiscovery(nil, "test", interval, refresh)

	ch := make(chan []*targetgroup.Group)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go d.Run(ctx, ch)

	tg := <-ch
	require.Equal(t, tg1, tg)

	tg = <-ch
	require.Equal(t, tg2, tg)

	failureCount, err := getCounterValue(d.failures)
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, float64(1), failureCount)
}

func TestSkipRefresh(t *testing.T) {
	resetMetrics()
	tg1 := []*targetgroup.Group{
		{
			Source: "tg",
			Targets: []model.LabelSet{
				{
					model.LabelName("t1"): model.LabelValue("v1"),
				},
			},
			Labels: model.LabelSet{
				model.LabelName("l1"): model.LabelValue("lv1"),
			},
		},
	}
	tg2 := []*targetgroup.Group{
		{
			Source: "tg",
		},
	}

	var i int
	refresh := func(ctx context.Context) ([]*targetgroup.Group, error) {
		i++
		switch i {
		case 1:
			return tg1, nil
		case 2:
			return nil, fmt.Errorf("up to date: %w", ErrSkipUpdate)
		}
		return tg2, nil
	}
	interval := time.Millisecond
	d := NewDiscovery(nil, "test", interval, refresh)

	ch := make(chan []*targetgroup.Group)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go d.Run(ctx, ch)

	tg := <-ch
	require.Equal(t, tg1, tg)

	tg = <-ch
	require.Equal(t, tg2, tg)

	failureCount, err := getCounterValue(d.failures)
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, float64(0), failureCount)
}

func getCounterValue(metric prometheus.Metric) (float64, error) {
	var m = &dto.Metric{}
	if err := metric.Write(m); err != nil {
		return 0, err
	}
	return m.Counter.GetValue(), nil
}

func resetMetrics() {
	failuresCount.Reset()
	duration.Reset()
}
