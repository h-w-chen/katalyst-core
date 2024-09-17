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

import (
	"fmt"

	"github.com/spf13/afero"
)

func getThreads(fs afero.Fs, pid int) ([]string, error) {
	taskFolder := fmt.Sprintf(tmplProcTaskFolder, pid)
	infos, err := afero.ReadDir(fs, taskFolder)
	if err != nil {
		return nil, err
	}

	tids := make([]string, len(infos))
	for i, info := range infos {
		if !info.IsDir() {
			continue
		}

		tids[i] = info.Name()
	}

	return tids, nil
}
