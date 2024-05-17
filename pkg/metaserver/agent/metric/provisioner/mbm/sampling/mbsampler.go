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

type mbSampler struct{}

func (m mbSampler) Shutdown() {
	//TODO implement me
	panic("implement me")
}

func (m mbSampler) Init() {
}

func (m mbSampler) Sample(ctx context.Context) {
	//TODO implement me
	panic("implement me")
}

func NewMBSampler(metricConf *machine.KatalystMachineInfo) MBSampler {
	return &mbSampler{}
}
