package advisor

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/kubewharf/katalyst-core/pkg/util/general"
)

const (
	intervalRunOnce = time.Second * 3
)

type Advisor interface {
	Init() error
	Run(ctx context.Context)
}

type gpuAdvisor struct{}

func (g *gpuAdvisor) Init() error {
	// todo: build gpu power reader

	//TODO implement me
	panic("implement me")
}

func (g *gpuAdvisor) Run(ctx context.Context) {
	if err := g.start(); err != nil {
		general.Errorf("pap-gpu: failed to start gpu advisor: %v", err)
		return
	}

	defer g.close()

	wait.Until(func() { g.run(ctx) }, intervalRunOnce, ctx.Done())
	general.Infof("pap-gpu: advisor Run exited")
}

func (g *gpuAdvisor) run(ctx context.Context) {
	general.InfofV(6, "pap-gpu: run once begin")

	general.InfofV(6, "pap-gpu: run once end")
}

func (g *gpuAdvisor) start() error {
	// to impl
	return nil
}

func (g *gpuAdvisor) close() {
	// to impl
}

func New() (Advisor, error) {
	return &gpuAdvisor{}, nil
}
