package gcl

import "golang.org/x/sys/unix"

// systemAvailableMemory approximates the available memory on macOS. Free
// page counts are misleading there (the kernel keeps memory filled with
// file cache and compressed pages), so half of the physical memory is
// treated as available.
func systemAvailableMemory() uint64 {
	total, err := unix.SysctlUint64("hw.memsize")
	if err != nil {
		return 0
	}
	return total / 2
}
