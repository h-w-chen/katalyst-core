package mbm

import (
	"context"
	"testing"
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
	m := NewMBMetricsProvisioner(nil, nil, nil, nil, nil)
	m.(*MBMetricsProvisioner).sampleFunc = stub.foo

	m.Run(context.TODO())

	if !stub.sampled {
		t.Errorf("expected Run ended being sampled (once), but not")
	}
}
