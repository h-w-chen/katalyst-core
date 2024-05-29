package mbw

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/spf13/afero"
)

var (
	instanceTest *afero.Afero
	onceTest     sync.Once
)

// MUST not be called in prod code!
// for unit test only.
func SetupTestFileSystem() {
	onceTest.Do(func() {
		instanceTest = &afero.Afero{Fs: afero.NewMemMapFs()}

		// we would like to have below device files exist for testing
		fakeFiles := []struct {
			dir     string
			file    string
			content string
		}{
			{
				dir:     "/sys/devices/system/node/node0/cpu0/cache/index3/",
				file:    "shared_cpu_list",
				content: "0-1\n",
			},
			{
				dir:     "/sys/devices/system/node/node0/cpu1/cache/index3/",
				file:    "shared_cpu_list",
				content: "0-1\n",
			},
			{
				dir:     "/sys/devices/system/node/node1/cpu2/cache/index3/",
				file:    "shared_cpu_list",
				content: "2-3\n",
			},
			{
				dir:     "/sys/devices/system/node/node1/cpu3/cache/index3/",
				file:    "shared_cpu_list",
				content: "2-3\n",
			},
		}

		for _, entry := range fakeFiles {
			instanceTest.MkdirAll(entry.dir, os.ModePerm)
			instanceTest.WriteFile(filepath.Join(entry.dir, entry.file), []byte(entry.content), os.ModePerm)
		}
	})
	appOS = instanceTest
}
