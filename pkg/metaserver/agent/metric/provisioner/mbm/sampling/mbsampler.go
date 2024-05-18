package sampling

import (
	"context"

	"github.com/kubewharf/katalyst-core/pkg/util/machine"
)

type MBSampler interface {
	Init()
	Shutdown()
	Sample(context.Context)
}

type MBSamplerFactory func(metricConf *machine.KatalystMachineInfo) MBSampler

type mbSampler struct {
	monitor *MBMonitor
}

func (m *mbSampler) Shutdown() {
	//TODO implement me
	panic("implement me")
}

func (m *mbSampler) Init() {
	if err := m.monitor.Init(); err != nil {
		// todo: log error
		// to set no-op monitor to avoid malfunctions
		m.monitor = nil
	}
}

func (m *mbSampler) Sample(ctx context.Context) {
	if m.monitor == nil {
		return
	}

	if err := m.monitor.ServePackageMB(); err == nil {
		// todo: save to metrics store
		//m.monitor.MemoryBandwidth.Packages
	}

	if err := m.monitor.ServeCoreMB(); err == nil {
		// todo: save fake numa stats
		// m.monitor.MemoryBandwidth.Numas
	}
}

func NewMBSampler(metricConf *machine.KatalystMachineInfo) MBSampler {
	monitor, err := NewMonitor()
	if err != nil {
		// todo: log error
		// not to crash as mb monitor is not the most essential; fine keeping whole machine functioning with reduced capacity
		return nil
	}

	return &mbSampler{
		monitor: monitor,
	}
}
