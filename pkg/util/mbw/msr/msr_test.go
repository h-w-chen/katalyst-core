package msr

import (
	"testing"
)

func TestMSRDev_Close(t *testing.T) {
	t.Parallel()

	// set up test stub
	SetupTestSyscaller()

	testMSRDev := MSRDev{fd: 5}
	if err := testMSRDev.Close(); err != nil {
		t.Errorf("expcted no error, got %#v", err)
	}
}

func TestMSR(t *testing.T) {
	t.Parallel()

	// set up test stub
	SetupTestSyscaller()

	_, err := MSR(9)
	if err != nil {
		t.Errorf("unexpected error: %#v", err)
	}
}
