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

package mbm

import (
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/kubewharf/katalyst-api/pkg/plugins/skeleton"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/resctrl"
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/resctrl/mbm/raw"
	"github.com/kubewharf/katalyst-core/pkg/util/general"
)

// intervalNumaNodeMBM is the interval to scan numa node mon_data files (of its CCDs') and calculate the MB
const intervalNumaNodeMBM = time.Second * 1

type NodeMonitor interface {
	GetMB(node int) map[int]int
	skeleton.GenericPlugin
}

type nodeMonitor struct {
	// table of numa nodes' CCDs
	ccdByNode map[int][]int
	// it delegates the calculation to mb calculator
	mbCalculator raw.MBCalculator
	// it may need help from task manager to get active tasks' mb in order to aggregate whole node's mb
	taskMonitor TaskMonitor

	rwLock sync.RWMutex
	// in mem db to save ccd's mb, calculated by background goroutine
	mbCCD map[int]int

	// signal to stop the background goroutine
	chStop chan struct{}
}

func (n *nodeMonitor) Name() string {
	return "numa_node_mb_monitor"
}

func (n *nodeMonitor) Start() error {
	general.Infof("mbm: numa node resctrl mbm monitor started")
	go n.run()
	return nil
}

func (n *nodeMonitor) updateMB(ccd, mb int) {
	n.rwLock.Lock()
	defer n.rwLock.Unlock()
	n.mbCCD[ccd] = mb
}

func (n *nodeMonitor) run() {
	general.Infof("mbm: numa node resctrl mbm monitor main loop run")
	wait.Until(n.scan, intervalNumaNodeMBM, n.chStop)
	general.Infof("mbm: numa node resctrl mbm monitor run exited")
}

func (n *nodeMonitor) scan() {
	// to scan all root level ctrl groups' mon data wrt relevant CCDs
	for node, ccds := range n.ccdByNode {
		ctrlGroupFolder := resctrl.GetNodeMBAFolder(node)
		mbs := n.mbCalculator.CalcMB(ctrlGroupFolder, ccds)
		mbs_tasks := n.taskMonitor.AggregateNodeMB(node)
		for ccd, mb := range mbs {
			// only update when valid data is collected
			if mb != raw.InvalidMB && mbs_tasks[ccd] != raw.InvalidMB {
				n.updateMB(ccd, mb+mbs_tasks[ccd])
			}
		}
	}
}

func (n *nodeMonitor) Stop() error {
	general.Infof("mbm: numa node resctrl mbm monitor being requested to stop")
	n.chStop <- struct{}{}
	return nil
}

func (n *nodeMonitor) GetMB(node int) map[int]int {
	n.rwLock.RLock()
	defer n.rwLock.RUnlock()
	return n.getMB(node)
}

func (n *nodeMonitor) getMB(node int) map[int]int {
	result := make(map[int]int)
	for _, ccds := range n.ccdByNode {
		for _, ccd := range ccds {
			result[ccd] = n.mbCCD[ccd]
		}
	}
	return result
}

func NewNodeMonitor(ccdByNode map[int][]int, calc raw.MBCalculator, taskMonitor TaskMonitor) (NodeMonitor, error) {
	return &nodeMonitor{
		ccdByNode:    ccdByNode,
		mbCalculator: calc,
		taskMonitor:  taskMonitor,
		mbCCD:        make(map[int]int),
		chStop:       make(chan struct{}),
	}, nil
}
