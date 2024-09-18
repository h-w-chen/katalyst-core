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

import (
	"errors"
	"fmt"
	"k8s.io/apimachinery/pkg/util/sets"
)

// PoolsPackage aggregates multiple app poolsByNode within the impact scope
type PoolsPackage interface {
	GetID() int
	GetMode() MBAllocationMode

	GetAppPools() []AppPool
	AddAppPool(nodes []int) (AppPool, error)
	DeleteAppPool(pool AppPool) error
}

type poolsPackage struct {
	id         int
	nodes      sets.Int
	ccdsByNode map[int]sets.Int

	mode        MBAllocationMode
	poolsByNode map[int]AppPool
	pools       []AppPool
}

func (p *poolsPackage) AddAppPool(nodes []int) (AppPool, error) {
	if !p.nodes.HasAll(nodes...) {
		return nil, errors.New("nodes not all in package scope")
	}

	for _, node := range nodes {
		if _, ok := p.poolsByNode[node]; ok {
			return nil, fmt.Errorf("node %d is busy by other app pool", node)
		}
	}

	pool := p.createAppPool(nodes)
	return pool, nil
}

func (p *poolsPackage) createAppPool(nodes []int) AppPool {
	ccds := make(map[int][]int)
	for _, node := range nodes {
		if _, ok := ccds[node]; !ok {
			ccds[node] = make([]int, 0)
		}
		for ccd, _ := range p.ccdsByNode[node] {
			ccds[node] = append(ccds[node], ccd)
		}
	}

	pool := &appPool{
		packageID: p.GetID(),
		numaNodes: nodes,
		ccds:      ccds,
	}

	for _, node := range nodes {
		p.poolsByNode[node] = pool
	}

	p.pools = append(p.pools, pool)

	return pool
}

func (p *poolsPackage) DeleteAppPool(pool AppPool) error {
	// todo: more stringent check and return errors
	var targetInPools AppPool
	nodesInPool := pool.GetNUMANodes()
	for _, node := range nodesInPool {
		targetInPools = p.poolsByNode[node]
		delete(p.poolsByNode, node)
	}

	newPools := make([]AppPool, 0)
	for _, p := range p.pools {
		if p != targetInPools {
			newPools = append(newPools, p)
		}
	}
	p.pools = newPools
	return nil
}

func (p *poolsPackage) GetID() int {
	return p.id
}

func (p *poolsPackage) GetMode() MBAllocationMode {
	return p.mode
}

func (p *poolsPackage) GetAppPools() []AppPool {
	return p.pools
}

func newPackage(id int, nodes []int, ccdsByNode map[int]sets.Int) PoolsPackage {
	setNodes := make(sets.Int)
	for _, n := range nodes {
		setNodes.Insert(n)
	}

	ccdLookup := make(map[int]sets.Int)
	for node, ccds := range ccdsByNode {
		if !setNodes.Has(node) {
			continue
		}
		for ccd, _ := range ccds {
			if ccdLookup[node] == nil {
				ccdLookup[node] = make(sets.Int)
			}
			ccdLookup[node].Insert(ccd)
		}
	}

	return &poolsPackage{
		id:          id,
		nodes:       setNodes,
		ccdsByNode:  ccdLookup,
		poolsByNode: make(map[int]AppPool),
	}
}
