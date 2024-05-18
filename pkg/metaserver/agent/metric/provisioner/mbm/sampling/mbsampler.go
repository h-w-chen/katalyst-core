package sampling

import (
	"context"

	"github.com/kubewharf/katalyst-core/pkg/metaserver/agent/metric/provisioner/mbm/sampling/mbwmanager"
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

type MBMonitor interface {
	Init() error
	Stop()
	ServePackageMB() error
	ServeCoreMB() error
	GetPackageSamples([]float64) []float64
	GetNUMASamples([]float64) []float64
}

type mbSampler struct {
	monitor MBMonitor

	packageTotalMBs []float64
	numaTotalMBs    []float64

	packageSampleWriter SampleWriter
	numaSampleWriter    SampleWriter
}

func (m *mbSampler) Shutdown() {
	if m.monitor == nil {
		return
	}

	m.monitor.Stop()
}

func (m *mbSampler) Init() {
	if err := m.monitor.Init(); err != nil {
		// todo: log error
		// to set no-op monitor to avoid malfunctions
		m.monitor = nil
	}
}

func (m *mbSampler) Sample(ctx context.Context) {
	select {
	case <-ctx.Done():
		// clean up as graceful termination
		m.Shutdown()
		return
	default:
		m.sample(ctx)
	}
}

func (m *mbSampler) sample(ctx context.Context) {
	if m.monitor == nil {
		return
	}

	if err := m.monitor.ServePackageMB(); err == nil {
		if samples := m.monitor.GetPackageSamples(m.packageTotalMBs); samples != nil {
			m.packageSampleWriter.Write(samples)
		}
	}
	//else {
	//	// todo: log error
	//}

	if err := m.monitor.ServeCoreMB(); err == nil {
		if samples := m.monitor.GetNUMASamples(m.numaTotalMBs); samples != nil {
			m.numaSampleWriter.Write(samples)
		}
	}
	//else {
	//	// todo: log error
	//}
}

func NewMBSampler(metricConf *machine.KatalystMachineInfo, packageSampleWriter, numaSampleWriter SampleWriter) MBSampler {
	// todo: revise to accept machine info as param
	monitor, err := mbwmanager.NewMonitor()
	if err != nil {
		// todo: log error
		// not to crash as mb monitor is not the most essential; fine keeping whole machine functioning with reduced capacity
		return nil
	}

	return &mbSampler{
		monitor:             monitor,
		packageTotalMBs:     make([]float64, monitor.NumPackages),
		numaTotalMBs:        make([]float64, monitor.NumNUMANodes),
		packageSampleWriter: packageSampleWriter,
		numaSampleWriter:    numaSampleWriter,
	}
}
