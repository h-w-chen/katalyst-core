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

package mba

// mbaGroup keeps the MBAs (numa nodes) that share one MB package
type mbaGroup map[int]*MBA

// MBAPackage puts the MBA control-groups that share NPS* memory bandwidth resources in one slot
// e.g. in NPS1 machine with 4 numa per socket, numa nodes 0-3 in package 0, 4-7 package 1
type MBAPackage map[int]mbaGroup
