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

const (
	FsRoot     = "/sys/fs/resctrl"
	FolderPerm = 0755
	FilePerm   = 0644

	// NumaFolderPrefix makes the folder like "node_X"
	NumaFolderPrefix = "node_"

	CPUListFile  = "cpus_list"
	SchemataFile = "schemata"

	MonGroupRoot = "/sys/fs/resctrl/mon_groups"
	TasksFile    = "tasks"
	MBRawFile    = "mbm_total_bytes"
)
