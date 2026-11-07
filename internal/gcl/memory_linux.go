package gcl

import (
	"bufio"
	"os"
	"path"
	"strconv"
	"strings"
)

// systemAvailableMemory returns MemAvailable from /proc/meminfo, lowered
// to the headroom left by the cgroup (v2) memory limits of this process
// and its ancestors, if any is tighter.
func systemAvailableMemory() uint64 {
	available := memAvailable("/proc/meminfo")
	if headroom, ok := cgroupHeadroom("/proc/self/cgroup", "/sys/fs/cgroup"); ok {
		if available == 0 || headroom < available {
			available = headroom
		}
	}
	return available
}

// memAvailable parses the MemAvailable line (in kB) of a meminfo file.
func memAvailable(meminfoPath string) uint64 {
	f, err := os.Open(meminfoPath)
	if err != nil {
		return 0
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		rest, ok := strings.CutPrefix(scanner.Text(), "MemAvailable:")
		if !ok {
			continue
		}
		kb, err := strconv.ParseUint(strings.TrimSuffix(strings.TrimSpace(rest), " kB"), 10, 64)
		if err != nil {
			return 0
		}
		return kb * 1024
	}
	return 0
}

// cgroupHeadroom returns the smallest memory.max - memory.current along the
// cgroup v2 hierarchy of this process. ok is false if no limit is set.
func cgroupHeadroom(procCgroupPath, cgroupRoot string) (uint64, bool) {
	data, err := os.ReadFile(procCgroupPath)
	if err != nil {
		return 0, false
	}

	var group string
	for line := range strings.SplitSeq(string(data), "\n") {
		if rest, ok := strings.CutPrefix(line, "0::"); ok {
			group = rest
			break
		}
	}
	if group == "" {
		return 0, false
	}

	var headroom uint64
	found := false
	for dir := path.Clean("/" + group); ; dir = path.Dir(dir) {
		limit, okLimit := readCgroupUint(path.Join(cgroupRoot, dir, "memory.max"))
		current, okCurrent := readCgroupUint(path.Join(cgroupRoot, dir, "memory.current"))
		if okLimit && okCurrent {
			h := uint64(0)
			if limit > current {
				h = limit - current
			}
			if !found || h < headroom {
				headroom = h
				found = true
			}
		}
		if dir == "/" {
			break
		}
	}
	return headroom, found
}

// readCgroupUint reads a single number from a cgroup file; "max" (no
// limit) and unreadable files report ok == false.
func readCgroupUint(file string) (uint64, bool) {
	data, err := os.ReadFile(file)
	if err != nil {
		return 0, false
	}
	v, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}
