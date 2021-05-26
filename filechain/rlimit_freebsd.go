// +build freebsd

package filechain

import "syscall"

func setNoFile(value uint64) error {
	rLimit := syscall.Rlimit{
		Cur: int64(value),
		Max: int64(value),
	}
	return syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
}
