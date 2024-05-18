package mbm

import (
	"context"
	"testing"

	"github.com/kubewharf/katalyst-core/pkg/config/agent/metaserver"
	"github.com/kubewharf/katalyst-core/pkg/metaserver/agent/metric/provisioner/mbm/sampling"
	"github.com/kubewharf/katalyst-core/pkg/util/machine"
)

type stubSampler struct {
	initialized bool
	sampled     bool
}

func (d *stubSampler) Init() {
	d.initialized = true
}

func (d *stubSampler) Shutdown() {
}

func (d *stubSampler) Sample(_ context.Context) {
	d.sampled = true
}

func TestMBMetricsProvisioner_New_to_Init_Run_to_sample(t *testing.T) {
	t.Parallel()

	stub := &stubSampler{}
	metricConf := &metaserver.MetricConfiguration{
		MBMetricConfiguration: &metaserver.MBMetricConfiguration{
			MachineInfo: &machine.KatalystMachineInfo{},
			SamplerFactory: func(*machine.KatalystMachineInfo, sampling.SampleWriter, sampling.SampleWriter) sampling.MBSampler {
				return stub
			},
		},
	}
	m := NewMBMetricsProvisioner(nil, metricConf, nil, nil, nil)

	if !stub.initialized {
		t.Errorf("expected New to initialize sampler, but not")
	}

	m.Run(context.TODO())

	if !stub.sampled {
		t.Errorf("expected Run ended being sampled (once), but not")
	}
}
