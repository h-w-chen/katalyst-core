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

package reader

import (
	"bytes"
	"context"
	"os/exec"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type PowerReader interface {
	Start() error
	GetTotalPower(ctx context.Context) (int, error)
}

// execFunc is the function signature for executing commands, swappable for testing.
type execFunc func(name string, arg ...string) ([]byte, error)

// todo: replace with a more efficient gpu reader
type nvidiaSmiPowerReader struct {
	execCommand execFunc
}

func (n *nvidiaSmiPowerReader) Start() error {
	// verify nvidia-smi is available and can query GPU power
	_, err := n.execCommand("nvidia-smi", "--query-gpu=power.draw", "--format=csv,noheader,nounits")
	if err != nil {
		return errors.Wrap(err, "nvidia-smi is not available")
	}
	return nil
}

func (n *nvidiaSmiPowerReader) GetTotalPower(ctx context.Context) (int, error) {
	output, err := n.execCommand("nvidia-smi", "--query-gpu=power.draw", "--format=csv,noheader,nounits")
	if err != nil {
		return 0, errors.Wrap(err, "failed to query gpu power via nvidia-smi")
	}

	return parseTotalPower(output)
}

// parseTotalPower parses the output of nvidia-smi power query.
// Each line contains a float value in watts for one GPU; returns the sum across all GPUs.
func parseTotalPower(output []byte) (int, error) {
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	total := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		val, err := strconv.ParseFloat(line, 64)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to parse gpu power value: %s", line)
		}
		total += int(val)
	}
	return total, nil
}

func defaultExec(name string, arg ...string) ([]byte, error) {
	cmd := exec.Command(name, arg...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return stdout.Bytes(), nil
}

func New() PowerReader {
	return &nvidiaSmiPowerReader{
		execCommand: defaultExec,
	}
}
