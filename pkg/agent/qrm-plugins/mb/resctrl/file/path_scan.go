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

package file

import (
	"path"

	"github.com/spf13/afero"

	resctrlconsts "github.com/kubewharf/katalyst-core/pkg/agent/qrm-plugins/mb/resctrl/consts"
)

func getResctrlSubMonGroups(fs afero.Fs, ctrlGroup string) ([]string, error) {
	result := make([]string, 0)
	dir := path.Join(resctrlconsts.FsRoot, ctrlGroup, resctrlconsts.SubGroupMonRoot)
	fis, err := afero.ReadDir(fs, dir)
	if err != nil {
		return nil, err
	}

	for _, fi := range fis {
		if !fi.IsDir() {
			continue
		}
		monGroup := fi.Name()
		result = append(result, path.Join(dir, monGroup))
	}

	return result, nil
}

func GetResctrlMonGroups(fs afero.Fs) ([]string, error) {
	result := make([]string, 0)
	fis, err := afero.ReadDir(fs, resctrlconsts.FsRoot)
	if err != nil {
		return nil, err
	}

	for _, fi := range fis {
		if !fi.IsDir() {
			continue
		}
		qosLevel := fi.Name()
		monGroupPaths, err := getResctrlSubMonGroups(fs, qosLevel)
		if err != nil {
			return nil, err
		}
		result = append(result, monGroupPaths...)
	}
	return result, nil
}
