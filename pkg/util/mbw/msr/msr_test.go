package msr

import (
	"sync"
	"testing"
)

var (
	instanceTest syscaller
	onceTest     sync.Once
)

type stubSyscaller struct{}

func (s stubSyscaller) Close(fd int) (err error) {
	return nil
}

func (s stubSyscaller) Open(path string, mode int, perm uint32) (fd int, err error) {
	return 99, nil
}

var _ syscaller = &stubSyscaller{}

func TestMSRDev_Close(t *testing.T) {
	t.Parallel()

	// set up test stub
	AppSyscall = func() syscaller {
		onceTest.Do(func() {
			instanceTest = &stubSyscaller{}
		})
		return instanceTest
	}()

	testMSRDev := MSRDev{fd: 5}
	if err := testMSRDev.Close(); err != nil {
		t.Errorf("expcted no error, got %#v", err)
	}
}

func TestMSR(t *testing.T) {
	t.Parallel()

	// set up test stub
	AppSyscall = func() syscaller {
		onceTest.Do(func() {
			instanceTest = &stubSyscaller{}
		})
		return instanceTest
	}()

	_, err := MSR(9)
	if err != nil {
		t.Errorf("unexpected error: %#v", err)
	}
}
