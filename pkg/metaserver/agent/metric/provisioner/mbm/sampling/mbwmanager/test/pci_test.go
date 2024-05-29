package test

import (
	"testing"

	"github.com/kubewharf/katalyst-core/pkg/metaserver/agent/metric/provisioner/mbm/sampling/mbwmanager"
	"github.com/kubewharf/katalyst-core/pkg/util/machine"
	"github.com/kubewharf/katalyst-core/pkg/util/mbw/msr"
)

func Test_InitPCIAccess(t *testing.T) {
	t.Parallel()
	msr.SetupTestSyscaller()

	mbm := mbwmanager.MBMonitor{
		SysInfo: &mbwmanager.SysInfo{
			KatalystMachineInfo: machine.KatalystMachineInfo{
				CPUTopology: &machine.CPUTopology{
					NumCPUs:      4,
					NumCores:     4,
					NumSockets:   1,
					NumNUMANodes: 2,
					CPUDetails: machine.CPUDetails{
						0: machine.CPUInfo{
							NUMANodeID: 0,
							SocketID:   0,
							CoreID:     0,
						},
						1: machine.CPUInfo{
							NUMANodeID: 0,
							SocketID:   0,
							CoreID:     1,
						},
						2: machine.CPUInfo{
							NUMANodeID: 1,
							SocketID:   0,
							CoreID:     2,
						},
						3: machine.CPUInfo{
							NUMANodeID: 1,
							SocketID:   0,
							CoreID:     3,
						},
					},
				},
			},
			Vendor: mbwmanager.CPU_VENDOR_AMD,
			Family: 0x19,
			Model:  mbwmanager.AMD_ZEN4_GENOA_B,
		},
	}

	if err := mbm.InitPCIAccess(); err != nil {
		t.Errorf("unexpected error %#v", err)
	}
}
