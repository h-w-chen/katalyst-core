package msr

import (
	"sync"
	"syscall"
)

var (
	AppSyscall Syscaller = &realOS{}
)

type Syscaller interface {
	Close(fd int) (err error)
	Open(path string, mode int, perm uint32) (fd int, err error)
	Pread(fd int, p []byte, offset int64) (n int, err error)
	Pwrite(fd int, p []byte, offset int64) (n int, err error)
}

type realOS struct{}

func (r realOS) Open(path string, mode int, perm uint32) (fd int, err error) {
	return syscall.Open(path, mode, perm)
}

func (r realOS) Close(fd int) (err error) {
	return syscall.Close(fd)
}

func (r realOS) Pread(fd int, p []byte, offset int64) (n int, err error) {
	return syscall.Pread(fd, p, offset)
}

func (r realOS) Pwrite(fd int, p []byte, offset int64) (n int, err error) {
	return syscall.Pwrite(fd, p, offset)
}

var _ Syscaller = &realOS{}

// note: below are for testing only; MUST not be used in prod code
var (
	instanceTest Syscaller
	onceTest     sync.Once
)

func SetupTestSyscaller() {
	onceTest.Do(func() {
		instanceTest = &stubSyscaller{}
	})
	AppSyscall = instanceTest
}

type stubSyscaller struct{}

func (s stubSyscaller) Pwrite(fd int, p []byte, offset int64) (n int, err error) {
	return 8, nil
}

func (s stubSyscaller) Pread(fd int, p []byte, offset int64) (n int, err error) {
	p[7] = 0x003B //59
	return 8, nil
}

func (s stubSyscaller) Close(fd int) (err error) {
	return nil
}

func (s stubSyscaller) Open(path string, mode int, perm uint32) (fd int, err error) {
	return 99, nil
}

var _ Syscaller = &stubSyscaller{}
