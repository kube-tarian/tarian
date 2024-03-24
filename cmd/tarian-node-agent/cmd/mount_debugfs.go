package cmd

import (
	"syscall"
)

// DebugFSMagic indicates statfs result is a DebugFS filesystem
//
// https://man7.org/linux/man-pages/man2/statfs.2.html
const DebugFSMagic = 0x64626720

// DebugFSRoot is the location of the DebugFS filesystem
const DebugFSRoot = "/sys/kernel/debug"

func isDebugFsMounted() bool {
	b := syscall.Statfs_t{}
	err := syscall.Statfs(DebugFSRoot, &b)
	if err != nil {
		return false
	}

	return b.Type == DebugFSMagic
}

func mountDebugFs() error {
	return syscall.Mount(DebugFSRoot, DebugFSRoot, "debugfs", 0, "")
}
