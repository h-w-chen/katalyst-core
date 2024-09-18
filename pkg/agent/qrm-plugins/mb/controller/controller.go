/*
Copyright 2022 The Katalyst Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/allocator"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/controller/policy"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/monitor"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/numapackage"
	"github.com/kubewharf/katalyst-core/pkg/util/general"
)

const (
	intervalMBController = time.Second * 1
)

type Controller struct {
	mbMonitor   monitor.Monitor
	mbAllocator allocator.Allocator

	packageManager *numapackage.Manager
}

func (c Controller) Run(ctx context.Context) {
	wait.UntilWithContext(ctx, c.run, intervalMBController)
}

func (c Controller) run(ctx context.Context) {
	for _, p := range c.packageManager.GetPackages() {
		if p.GetMode() == numapackage.MBAllocationModeHardPreempt {
			c.preemptPackage(ctx, p)
			continue
		}

		c.adjustPackage(ctx, p)
	}
}

// preemptPackage is called if package is in "hard-limit" preemption phase
func (c Controller) preemptPackage(ctx context.Context, p numapackage.MBPackage) {
	allocs, err := policy.CalcPreemptAllocs(p.GetUnits(), c.mbMonitor)
	if err != nil {
		general.Warningf("mbm: failed to set hard limits for admitted units due to error %v", err)
		return
	}

	if err := c.SetMBAllocs(allocs); err != nil {
		general.Warningf("mbm: failed to set hard limits for package %d due to error %v", p.GetID(), err)
		return
	}

	for _, u := range p.GetUnits() {
		if u.GetLifeCyclePhase() == numapackage.UnitPhaseAdmitted &&
			u.GetTaskType() == numapackage.TaskTypeSOCKET {
			u.SetPhase(numapackage.UnitPhaseReserved)
		}
	}
}

// adjustPackage is called when package is in regular state other than "hard-limiting"
func (c Controller) adjustPackage(ctx context.Context, p numapackage.MBPackage) {
	allocs, err := policy.CalcSoftAllocs(p.GetUnits(), policy.TotalPackageMB, policy.SocketLoungeMB, c.mbMonitor)
	if err != nil {
		general.Errorf("mbm: failed to calc soft limits for package %d: %v", p.GetID(), err)
	}
	if err := c.SetMBAllocs(allocs); err != nil {
		general.Warningf("mbm: failed to set soft limits for package %d due to error %v", p.GetID(), err)
		return
	}
}

func (c Controller) SetMBAllocs(mbs []policy.MBUnitAlloc) error {
	for _, alloc := range mbs {
		for _, node := range alloc.Unit.GetNUMANodes() {
			ccdCurrs := c.mbMonitor.GetMB(node)
			ccdAllocs := policy.CcdDistributeMB(alloc.MBUpperBound, ccdCurrs)
			if err := c.mbAllocator.AllocateMB(node, ccdAllocs); err != nil {
				return err
			}
		}
	}

	return nil
}

func New() *Controller {
	return &Controller{
		mbMonitor:   nil,
		mbAllocator: nil,
	}
}
