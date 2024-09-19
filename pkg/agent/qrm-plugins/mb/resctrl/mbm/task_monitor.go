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

type TaskMonitor struct {
	nodeLookup map[int][]int
	ccdByNode  map[int][]int

	mbTask map[int]int
}

func (t TaskMonitor) GetMBByProcessID(pid int) int {
	panic("impl")
}

func (t TaskMonitor) AggregateNodeMB(node int) map[int]int {
	panic("impl")
}
