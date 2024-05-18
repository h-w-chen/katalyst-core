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

type SampleWriter interface {
	Write([]float64)
}

type SamplerWriteFunc func([]float64)

func (f SamplerWriteFunc) Write(samples []float64) {
	f(samples)
}

type MBSamplerFactory func(metricConf *machine.KatalystMachineInfo, packageSampleWriter, numaSampleWriter SampleWriter) MBSampler

type mbSampler struct {
	monitor *MBMonitor

	packageSampleWriter SampleWriter
	numaSampleWriter    SampleWriter
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
		//m.packetSampleWriter.Write(m.monitor.MemoryBandwidth.Packages)
	}

	if err := m.monitor.ServeCoreMB(); err == nil {
		// todo: save fake numa stats
		// m.monitor.MemoryBandwidth.Numas
	}
}

func NewMBSampler(metricConf *machine.KatalystMachineInfo, packageSampleWriter, numaSampleWriter SampleWriter) MBSampler {
	monitor, err := NewMonitor()
	if err != nil {
		// todo: log error
		// not to crash as mb monitor is not the most essential; fine keeping whole machine functioning with reduced capacity
		return nil
	}

	return &mbSampler{
		monitor:             monitor,
		packageSampleWriter: packageSampleWriter,
		numaSampleWriter:    numaSampleWriter,
	}
}
