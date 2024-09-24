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

package ccd

type otherQoSPolicy struct {
	maxMB int
}

func (o otherQoSPolicy) Distribute(upperlimitMB int, ccdMB map[int]int) map[int]int {
	panic("impl")
}

func NewOtherQoSPolicy(maxMB int) CCDMBDistributor {
	return &otherQoSPolicy{maxMB: maxMB}
}
