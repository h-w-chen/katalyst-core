package mbm

import (
	"context"
	"testing"

	"github.com/kubewharf/katalyst-core/pkg/config/agent/metaserver"
	"github.com/kubewharf/katalyst-core/pkg/util/machine"
)

type testStub struct {
	sampled    bool
	cancelFunc context.CancelFunc
}

func (d *testStub) foo(_ context.Context) {
	d.sampled = true
}

func TestMBMetricsProvisioner_Run_to_sample_once(t *testing.T) {
	t.Parallel()

	stub := &testStub{}

	metricConf := &metaserver.MetricConfiguration{
		MBMetricConfiguration: &metaserver.MBMetricConfiguration{
			MachineInfo: &machine.KatalystMachineInfo{},
		},
	}
	m := NewMBMetricsProvisioner(nil, metricConf, nil, nil, nil)
	m.(*MBMetricsProvisioner).sampleFunc = stub.foo

	m.Run(context.TODO())

	if !stub.sampled {
		t.Errorf("expected Run ended being sampled (once), but not")
	}
}
