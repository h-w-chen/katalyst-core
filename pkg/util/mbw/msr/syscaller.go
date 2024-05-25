package msr

import (
	"syscall"
)

var (
	AppSyscall syscaller = &realOS{}
)

type syscaller interface {
	Close(fd int) (err error)
	Open(path string, mode int, perm uint32) (fd int, err error)
}

type realOS struct{}

func (r realOS) Open(path string, mode int, perm uint32) (fd int, err error) {
	return syscall.Open(path, mode, perm)
}

func (r realOS) Close(fd int) (err error) {
	return syscall.Close(fd)
}
