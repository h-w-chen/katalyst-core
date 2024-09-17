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
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/monitor"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/numapackage"
	"github.com/kubewharf/katalyst-core/pkg/util/general"
)

const (
	intervalMBController = time.Second * 1

	TotalPackageMB         = 115_000 // 115 GB
	SOCKETReverseMBPerNode = 30_000  // 30 GB MB reserved for SOCKET app per numa node
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

func (c Controller) preemptPackage(ctx context.Context, p numapackage.MBPackage) {
	// reserve certain MB for the admitting unit
	var mbToReserve int
	for _, u := range p.GetUnits() {
		switch u.GetLifeCyclePhase() {
		case numapackage.UnitPhaseAdmitted:
			if u.GetTaskType() == numapackage.TaskTypeSOCKET {
				mbToReserve += len(u.GetNUMANodes()) * SOCKETReverseMBPerNode
			}
		case numapackage.UnitPhaseReserved:
			mbToReserve += len(u.GetNUMANodes()) * 10_000 // 10GB reservation per node
		}
	}

	mbToAllocate := TotalPackageMB - mbToReserve
	var mbInUse int
	for _, u := range p.GetUnits() {
		if u.GetLifeCyclePhase() == numapackage.UnitPhaseReserved {
			continue
		}

		for _, n := range u.GetNUMANodes() {
			mbs, err := c.mbMonitor.GetMB(n)
			if err != nil {
				general.Warningf("mbm: failed to get numa node %d MB usage: %v", n, err)
				return
			}

			for _, mb := range mbs {
				mbInUse += mb
			}
		}
	}
	_ = mbToAllocate
	// todo: allocate mbToAllocate properly to other units y=than in admit/reserved phase

	for _, u := range p.GetUnits() {
		if u.GetLifeCyclePhase() == numapackage.UnitPhaseAdmitted &&
			u.GetTaskType() == numapackage.TaskTypeSOCKET {
			u.SetPhase(numapackage.UnitPhaseReserved)
		}
	}
	// todo: allocate reserved MB to units of reserved phase
}

func (c Controller) adjustPackage(ctx context.Context, p numapackage.MBPackage) {
	panic("impl")
}

func New() *Controller {
	return &Controller{
		mbMonitor:   nil,
		mbAllocator: nil,
	}
}
