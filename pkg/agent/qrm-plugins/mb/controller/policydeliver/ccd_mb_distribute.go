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

package policydeliver

import (
	"github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/controller/policy"
)

func distributeCCDMBs(total int, mbCCD map[int]int) map[int]int {
	currMB := 0
	for _, v := range mbCCD {
		currMB += v
	}

	// if all are in very small a fraction, treat them equally
	if currMB <= int(float64(total)*0.4) {
		ccdAllocs := make(map[int]int)
		for ccd, _ := range mbCCD {
			ccdAllocs[ccd] = total / len(mbCCD)
		}
		return ccdAllocs
	}

	ccdAllocs := make(map[int]int)
	for ccd, v := range mbCCD {
		ccdAllocs[ccd] = policy.ProrateAlloc(v, currMB, total)
	}

	return ccdAllocs
}
