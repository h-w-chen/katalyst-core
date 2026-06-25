package advisor

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/gpupoweraware/capper"
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/gpupoweraware/plan"
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/gpupoweraware/reader"
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
	general.Infof("pap-gpu: advisor Run")

	if err := g.start(); err != nil {
		general.Errorf("pap-gpu: advisor Run failed to start: %v", err)
		return
	}
	general.Infof("pap-gpu: advisor Run started")

	defer g.close()

	wait.Until(func() { g.runOnce(ctx) }, intervalRunOnce, ctx.Done())
	general.Infof("pap-gpu: advisor Run exited")
}

func (g *gpuAdvisor) runOnce(ctx context.Context) {
	general.InfofV(6, "pap-gpu: advisor: runOnce once begin")

	powerSpec, err := g.specFetcher.GetPowerSpec(ctx)
	if err != nil {
		general.Warningf("pap-gpu: advisor: failed to runOnce once: %v", err)
		return
	}

	totalPower, err := g.powerReader.GetTotalPower(ctx)
	if err != nil {
		general.Warningf("pap-gpu: advisor: failed to runOnce once: %v", err)
		return
	}

	powerPlan, err := g.planner.GetPlan(powerSpec, "default", totalPower)
	if err != nil {
		general.Warningf("pap-gpu: advisor: failed to runOnce once: %v", err)
		return
	}

	general.InfofV(6, "pap-gpu: advisor: get power powerSpec %v", *powerSpec)
	general.InfofV(6, "pap-gpu: advisor: get current total power %v", totalPower)
	general.InfofV(6, "pap-gpu: advisor: decide power plan %v", powerPlan)

	// todo: choose proper op level
	g.capper.CapWithLevel(ctx, capper.LevelAll, powerPlan.Target, totalPower)

	general.InfofV(6, "pap-gpu: advisor: runOnce once end")
}

func (g *gpuAdvisor) start() error {
	general.Infof("gpu-pap: gpuAdvisor.start() calling capper.Start()...")
	if err := g.capper.Start(); err != nil {
		general.Errorf("gpu-pap: capper.Start() failed: %v", err)
		return errors.Wrap(err, "power capper start failed")
	}
	general.Infof("gpu-pap: advisor: capper.Start() succeeded")

	general.Infof("gpu-pap: gpuAdvisor.start() calling powerReader.Start()...")
	if err := g.powerReader.Start(); err != nil {
		return errors.Wrap(err, "power reader start failed")
	}
	general.Infof("gpu-pap: advisor: powerReader.Start() succeeded")

	general.Infof("gpu-pap: gpuAdvisor.start() done")
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
