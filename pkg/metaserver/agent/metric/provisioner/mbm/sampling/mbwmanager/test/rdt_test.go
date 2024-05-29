package test

import (
	"github.com/kubewharf/katalyst-core/pkg/util/mbw/msr"
	"reflect"
	"testing"

	"github.com/kubewharf/katalyst-core/pkg/metaserver/agent/metric/provisioner/mbm/sampling/mbwmanager"
)

func Test_InitRDT(t *testing.T) {
	t.Parallel()
	msr.SetupTestSyscaller()

	mbm := mbwmanager.MBMonitor{
		SysInfo: &mbwmanager.SysInfo{
			Vendor:  mbwmanager.CPU_VENDOR_INTEL,
			DieSize: 2,                                   // dummy 2 numa nodes - so 2 CCD
			CCDMap:  map[int][]int{0: {0, 1}, 1: {2, 3}}, // dummy CCD for 2x2 numa
		},
		Controller: mbwmanager.MBController{RMIDMap: map[int]int{}},
	}
	if err := mbm.InitRDT(); err != nil {
		t.Errorf("unexpected error: %#v", err)
	}
}

func Test_Assign_Uniq_RMID(t *testing.T) {
	t.Parallel()
	msr.SetupTestSyscaller()

	mbm := mbwmanager.MBMonitor{
		SysInfo: &mbwmanager.SysInfo{
			Vendor: mbwmanager.CPU_VENDOR_AMD,
			Family: 0x19,
			Model:  mbwmanager.AMD_ZEN4_GENOA_B,
		},
	}
	if err := mbm.Assign_Uniq_RMID(1, 1234); err != nil {
		t.Errorf("unexpected error: %#v", err)
	}
}

func Test_ReadCoreMB(t *testing.T) {
	t.Parallel()
	msr.SetupTestSyscaller()

	want := map[mbwmanager.CORE_MB_EVENT_TYPE][]uint64{
		1: []uint64{0x3b00000000000000, 0x3b00000000000000},
		2: []uint64{0x3b00000000000000, 0x3b00000000000000},
	}
	mbm := mbwmanager.MBMonitor{
		SysInfo: &mbwmanager.SysInfo{
			MemoryBandwidth: mbwmanager.MemoryBandwidthInfo{
				Cores: []mbwmanager.CoreMB{{
					Package:    0,
					LRMB:       0,
					LRMB_Delta: 0,
					RRMB_Delta: 0,
					TRMB:       0,
					TRMB_Delta: 0,
				}, {
					Package:    1,
					LRMB:       1,
					LRMB_Delta: 1,
					RRMB_Delta: 1,
					TRMB:       1,
					TRMB_Delta: 1,
				}},
			}},
	}
	output, err := mbm.ReadCoreMB()

	if err != nil {
		t.Errorf("unexpected error: %#v", err)
	}
	if !reflect.DeepEqual(output, want) {
		t.Errorf("expected %#v, got %#v", want, output)
	}
}
