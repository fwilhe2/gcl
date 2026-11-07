//go:build !linux && !darwin && !windows

package gcl

// systemAvailableMemory is unknown on this platform; the default budget
// applies.
func systemAvailableMemory() uint64 {
	return 0
}
