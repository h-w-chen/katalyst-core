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
	"sync"
	"time"

	"github.com/spf13/afero"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/allocator"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/controller/mbdomain"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/controller/policy"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/monitor"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/qosgroup"
	resctrltask "github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/resctrl/task"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/task"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/task/cgcpuset"
	"github.com/kubewharf/katalyst-core/pkg/util/general"
)

const (
	intervalMBController = time.Second * 1
)

type Controller struct {
	cancel context.CancelFunc

	TaskManager *resctrltask.TaskManager
	cgCPUSet    *cgcpuset.CPUSet

	podMBMonitor    monitor.MBMonitor
	mbPlanAllocator allocator.PlanAllocator

	// exposure it for testability
	DomainManager *mbdomain.MBDomainManager

	policy policy.DomainMBPolicy

	chAdmit chan struct{}

	lock sync.RWMutex
	// expose below field to make test easier
	// todo: not to expose this field
	CurrQoSCCDMB map[qosgroup.QoSGroup]*monitor.MBQoSGroup
}

func (c *Controller) updateQoSCCDMB(qosCCDMB map[qosgroup.QoSGroup]*monitor.MBQoSGroup) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.CurrQoSCCDMB = qosCCDMB
}

func (c *Controller) GetDedicatedNodes() sets.Int {
	if tasks, err := c.TaskManager.GetQoSGroupedTask(qosgroup.QoSGroupDedicated); err == nil && len(tasks) > 0 {
		infoGetter := task.NewInfoGetter(c.cgCPUSet, tasks)
		dedicatedNodes := infoGetter.GetAssignedNumaNodes()
		general.InfofV(6, "mbm: identify dedicated pods numa nodes by cgroup mechanism: %v", dedicatedNodes)
		return dedicatedNodes
	}

	// fall back to educated guess by looking at the slots of active mb metrics
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.getDedicatedNodes()
}

// getDedicatedNodes identifies the nodes currently assigned to dedicated qos and having active traffic (including 0)
func (c *Controller) getDedicatedNodes() sets.Int {
	// identify the ccds having active traffic as dedicated groups
	dedicatedCCDMB, ok := c.CurrQoSCCDMB[qosgroup.QoSGroupDedicated]
	if !ok {
		return nil
	}

	dedicatedNodes := make(sets.Int)
	for ccd := range dedicatedCCDMB.CCDMB {
		node, err := c.DomainManager.GetNode(ccd)
		if err != nil {
			panic(err)
		}
		dedicatedNodes.Insert(node)
	}

	return dedicatedNodes
}

// ReqToAdjustMB requests controller to start a round of mb adjustment
func (c *Controller) ReqToAdjustMB() {
	select {
	case c.chAdmit <- struct{}{}:
	default:
	}
}

func (c *Controller) Run() {
	general.Infof("mbm: main control loop Run started")

	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel

	ticker := time.NewTicker(intervalMBController)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			general.Infof("mbm: main control loop Run exited")
			return
		case <-ticker.C:
			c.run(ctx)
		case <-c.chAdmit:
			general.InfofV(6, "mbm: process admit request")
			c.process(ctx)
		}
	}
}

func (c *Controller) run(ctx context.Context) {
	qosCCDMB, err := c.podMBMonitor.GetMBQoSGroups()
	if err != nil {
		general.Errorf("mbm: failed to get MB usages: %v", err)
	}

	c.updateQoSCCDMB(qosCCDMB)

	general.InfofV(6, "mbm: mb usage summary: %v", monitor.DisplayMBSummary(qosCCDMB))
	c.process(ctx)
}

func (c *Controller) process(ctx context.Context) {
	for i, domain := range c.DomainManager.Domains {
		// we only care about qosCCDMB manageable by the domain
		applicableQoSCCDMB := getApplicableQoSCCDMB(domain, c.CurrQoSCCDMB)
		general.InfofV(6, "mbm: domain %d mb stat: %#v", i, applicableQoSCCDMB)

		mbAlloc := c.policy.GetPlan(domain.MBQuota, domain, applicableQoSCCDMB)
		general.InfofV(6, "mbm: domain %d mb alloc plan: %v", i, mbAlloc)

		if err := c.mbPlanAllocator.Allocate(mbAlloc); err != nil {
			general.Errorf("mbm: failed to allocate mb plan for domain %d: %v", i, err)
		}
	}
}

func (c *Controller) Stop() error {
	if c.cancel == nil {
		return nil
	}

	c.cancel()
	return nil
}

func New(podMBMonitor monitor.MBMonitor, mbPlanAllocator allocator.PlanAllocator, domainManager *mbdomain.MBDomainManager, policy policy.DomainMBPolicy) (*Controller, error) {
	fs := afero.NewOsFs()
	return &Controller{
		podMBMonitor:    podMBMonitor,
		mbPlanAllocator: mbPlanAllocator,
		DomainManager:   domainManager,
		policy:          policy,
		chAdmit:         make(chan struct{}, 1),
		cgCPUSet:        cgcpuset.New(fs),
		TaskManager:     resctrltask.New(fs),
	}, nil
}

func getApplicableQoSCCDMB(domain *mbdomain.MBDomain, qosccdmb map[qosgroup.QoSGroup]*monitor.MBQoSGroup) map[qosgroup.QoSGroup]*monitor.MBQoSGroup {
	result := make(map[qosgroup.QoSGroup]*monitor.MBQoSGroup)

	for qos, mbQosGroup := range qosccdmb {
		for ccd, _ := range mbQosGroup.CCDs {
			if _, ok := mbQosGroup.CCDMB[ccd]; !ok {
				// no ccd-mb stat; skip it
				continue
			}
			if _, ok := domain.CCDNode[ccd]; ok {
				if _, ok := result[qos]; !ok {
					result[qos] = &monitor.MBQoSGroup{
						CCDs:  make(sets.Int),
						CCDMB: make(map[int]*monitor.MBData),
					}
				}
				result[qos].CCDs.Insert(ccd)
				result[qos].CCDMB[ccd] = qosccdmb[qos].CCDMB[ccd]
			}
		}
	}

	return result
}
