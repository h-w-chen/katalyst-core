package pci

import (
	"reflect"
	"testing"
)

func TestPCIDev_GetDevInfo(t *testing.T) {
	t.Parallel()
	var devTest = &PCIDev{
		vendor_id: 1988,
		device_id: 11,
		domain_16: 8,
		bus:       2,
		dev:       3,
		hdrtype:   1,
	}
	got := devTest.GetDevInfo()
	if 11 != got.DeviceID {
		t.Errorf("expected dev id 11, got %d", got.DeviceID)
	}
}

func TestPCIDev_BDFString(t *testing.T) {
	t.Parallel()
	var devTest = &PCIDev{
		vendor_id: 1988,
		device_id: 11,
		domain_16: 8,
		bus:       2,
		dev:       3,
		hdrtype:   1,
	}

	got := devTest.BDFString()
	want := "0000:02:03.0"
	if want != got {
		t.Errorf("expected %s, got %s", want, got)
	}
}

func TestPCIDev_GetDevNumaNode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		dev  *PCIDev
		want int
	}{
		{
			name: "negative path returns -1",
			dev: &PCIDev{
				vendor_id: 0xdead,
				device_id: 0xdead,
				domain_16: 8,
				bus:       0xde,
				dev:       3,
				hdrtype:   1,
			},
			want: -1,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.dev.GetDevNumaNode(); got != tt.want {
				t.Errorf("GetDevNumaNode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPCIDev_Init_Cleanup(t *testing.T) {
	t.Parallel()

	// this test does not verify specific attributes, but the general init/cleanup behavior able to run
	PCIDevInit()
	PCIDevCleanup()
}

func TestGetFirstIOHC(t *testing.T) {
	t.Parallel()

	testNode := 1
	testDevs := []*PCIDev{
		&PCIDev{
			vendor_id: 1988,
			device_id: 11,
			domain_16: 8,
			bus:       2,
			dev:       3,
			hdrtype:   1,
		},
	}

	var want *PCIDev = nil
	if got := GetFirstIOHC(testNode, testDevs); !reflect.DeepEqual(got, want) {
		t.Errorf("GetFirstIOHC() = %v, want %v", got, want)
	}
}
