package advisor

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/gpupoweraware/plan"
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/gpupoweraware/reader"
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/poweraware/capper"
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/poweraware/spec"
	"github.com/kubewharf/katalyst-core/pkg/util/general"
)

const (
	intervalRunOnce = time.Second * 3
)

type Advisor interface {
	Init() error
	Run(ctx context.Context)
}

type gpuAdvisor struct {
	specFetcher spec.SpecFetcher
	powerReader reader.PowerReader
	planner     plan.Planner
	capper      capper.PowerCapper
}

func (g *gpuAdvisor) Init() error {
	// todo: build gpu power reader
	return nil
}

func (g *gpuAdvisor) Run(ctx context.Context) {
	if err := g.start(); err != nil {
		general.Errorf("pap-gpu: advisor Run failed to start: %v", err)
		return
	}
	defer g.close()

	general.Infof("pap-gpu: advisor Run started")
	wait.Until(func() { g.run(ctx) }, intervalRunOnce, ctx.Done())
	general.Infof("pap-gpu: advisor Run exited")
}

func (g *gpuAdvisor) run(ctx context.Context) {
	general.InfofV(6, "pap-gpu: run once begin")

	powerSpec, err := g.specFetcher.GetPowerSpec(ctx)
	if err != nil {
		general.Warningf("pap-gpu: failed to run once: %v", err)
		return
	}

	totalPower, err := g.powerReader.GetTotalPower(ctx)
	if err != nil {
		general.Warningf("pap-gpu: failed to run once: %v", err)
		return
	}

	powerPlan, err := g.planner.GetPlan(powerSpec, "default", totalPower)
	if err != nil {
		general.Warningf("pap-gpu: failed to run once: %v", err)
		return
	}

	general.InfofV(6, "pap-gpu: get power powerSpec %v", *powerSpec)
	general.InfofV(6, "pap-gpu: get current total power %v", totalPower)
	general.InfofV(6, "pap-gpu: decide power plan %v", powerPlan)

	// todo: execute power plan via capper
	// g.capper.Cap()

	general.InfofV(6, "pap-gpu: run once end")
}

func (g *gpuAdvisor) start() error {
	if err := g.capper.Start(); err != nil {
		return errors.Wrap(err, "power capper start failed")
	}

	if err := g.powerReader.Start(); err != nil {
		return errors.Wrap(err, "power reader start failed")
	}

	return nil
}

func (g *gpuAdvisor) close() {
	// to impl
}

func New(specFetcher spec.SpecFetcher, powerCapper capper.PowerCapper) (Advisor, error) {
	return &gpuAdvisor{
		specFetcher: specFetcher,
		powerReader: reader.New(),
		planner:     plan.New(),
		capper:      powerCapper,
	}, nil
}
