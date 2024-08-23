/*
Copyright 2022 The Katalyst Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

//go:build integration
// +build integration

// integration test taking lengthy time; not to run as regular unit test
package power

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/poweraware/plugin"
	"github.com/kubewharf/katalyst-core/pkg/config"
	"github.com/kubewharf/katalyst-core/pkg/config/agent"
	"github.com/kubewharf/katalyst-core/pkg/config/agent/global"
	metricspool "github.com/kubewharf/katalyst-core/pkg/metrics/metrics-pool"
)

func TestPowerPressureEndpoint_RecoverService(t *testing.T) {
	t.Parallel()

	conf := &config.Configuration{
		AgentConfiguration: &agent.AgentConfiguration{
			GenericAgentConfiguration: &agent.GenericAgentConfiguration{
				PluginManagerConfiguration: &global.PluginManagerConfiguration{
					PluginRegistrationDir: "/tmp/test", //assuming no conflict
				},
			},
		},
	}

	dummyEmitter := metricspool.DummyMetricsEmitterPool{}.GetDefaultMetricsEmitter().WithTags("plugin-pap")
	ep := NewPowerPressureEndpoint(nil, nil, nil, nil, conf)
	name := ep.Name()
	assert.Equal(t, "node_power_pressure", name)

	// unavailable server; empty resp expected
	resp, err := ep.ThresholdMet(context.TODO())
	t.Logf("1st try: %#v", resp)
	t.Logf("1st error: %#v", err)
	assert.Equal(t, 0, int(resp.ThresholdValue), "server not live; expected 0 value")

	// starting a server, stuffing one evict, then meaningful resp is expected
	podEvictor, service, err := plugin.NewPowerPressureEvictPluginServer(conf, dummyEmitter)
	if err != nil {
		t.Errorf("failed to create pap eviction server: %v", err)
		return
	}
	podEvictor.Evict(context.TODO(), &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			UID: "1234",
		},
	})

	if err := service.Start(); err != nil {
		t.Errorf("failed to start pap eviction server: %v", err)
		return
	}

	resp, err = ep.ThresholdMet(context.TODO())
	t.Logf("2nd try: %#v", resp)
	t.Logf("2nd error: %#v", err)
	assert.Equal(t, 1, int(resp.ThresholdValue), "server is running; expected 1 value")
}
