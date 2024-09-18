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

package policy

import "github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/apppool"

const (
	TotalPackageMB = 116_000 // 116 GB

	SocketNodeMaxMB      = 60_000 // 60GBps max for socket (if one node)
	SocketNodeReservedMB = 35_000 // 35 GB MB reserved for SOCKET app per numa node
	SocketLoungeMB       = 6_000  // 6GB MB reserved as lounge size (ear marked for SOCKET pods overflow only)

	// todo: revise algo to enforce min MB
	NodeMinMB = 4_000 // each node at least 4GB to avoid starvation
)

// MBAlloc keeps the total MB allocated to an AppPool; it is the unit of MB adjustment produced by mb policy
type MBAlloc struct {
	AppPool      apppool.Pool
	MBUpperBound int // MB in MBps
}
