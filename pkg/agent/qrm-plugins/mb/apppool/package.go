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

package apppool

import "k8s.io/apimachinery/pkg/util/sets"

// PoolsPackage aggregates multiple app pools within the impact scope
type PoolsPackage interface {
	GetID() int
	GetMode() MBAllocationMode

	GetAppPools() []AppPool
	AddAppPool(nodes []int) (AppPool, error)
	DeleteAppPool(pool AppPool) error
}

type poolsPackage struct {
	id    int
	mode  MBAllocationMode
	nodes sets.Int
}

func (p poolsPackage) AddAppPool(nodes []int) (AppPool, error) {
	//TODO implement me
	panic("implement me")
}

func (p poolsPackage) DeleteAppPool(pool AppPool) error {
	//TODO implement me
	panic("implement me")
}

func (p poolsPackage) GetID() int {
	return p.id
}

func (p poolsPackage) GetMode() MBAllocationMode {
	return p.mode
}

func (p poolsPackage) GetAppPools() []AppPool {
	//TODO implement me
	panic("implement me")
}

func newPackage(id int) PoolsPackage {
	// todo: identify numa nodes based on die topology
	// for POC purpose, assuming numa nodes are id*4 - id*4 + 3
	nodes := make(sets.Int)
	for i := 0; i < 4; i++ {
		nodes.Insert(id*4 + i)
	}
	return &poolsPackage{
		id:    id,
		nodes: nodes,
	}
}
