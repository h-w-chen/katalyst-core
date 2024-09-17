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

package monitor

type Monitor interface {
	// GetMB gets the CCD's MB of the numa node
	GetMB(node int) (map[int]int, error)
}

type monitor struct{}

func (m monitor) GetMB(node int) (map[int]int, error) {
	// sum up all mon groups starting with "node_X_pid_*"
	// and ctrl group "node_X"
	panic("impl getMB")
}

func NewMonitor() Monitor {
	return &monitor{}
}
