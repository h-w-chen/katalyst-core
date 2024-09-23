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

package resctrl

type CtrlGroupMBSetter interface {
	SetMB(ctrlGroup string, ccdMB map[int]int) error
}

type ctrlGroupMBSetter struct {
	ccdMBSetter CCDMBSetter
}

func (c ctrlGroupMBSetter) SetMB(ctrlGroup string, ccdMB map[int]int) error {
	for ccd, mb := range ccdMB {
		if err := c.ccdMBSetter.SetMB(ctrlGroup, ccd, mb); err != nil {
			return err
		}
	}
	return nil
}

func NewCtrlGroupSetter() (CtrlGroupMBSetter, error) {
	return &ctrlGroupMBSetter{}, nil
}
