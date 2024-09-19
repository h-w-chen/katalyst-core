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

type NodeMonitor interface {
	GetMB(node int) map[int]int
}

type nodeMonitor struct {
	ccdByNode map[int][]int
	mbCCD     map[int]int

	taskMonitor *TaskMonitor
}

func (n nodeMonitor) GetMB(node int) map[int]int {
	result := make(map[int]int)
	for ccd, mb := range n.getMB(node) {
		result[ccd] += mb
	}
	for ccd, mb := range n.taskMonitor.AggregateNodeMB(node) {
		result[ccd] += mb
	}
	return result
}

func (n nodeMonitor) getMB(node int) map[int]int {
	result := make(map[int]int)
	for _, ccds := range n.ccdByNode {
		for _, ccd := range ccds {
			result[ccd] = n.mbCCD[ccd]
		}
	}
	return result
}

func NewNodeMonitor() NodeMonitor {
	return &nodeMonitor{}
}
