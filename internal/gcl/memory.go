package gcl

import (
	"fmt"
	"runtime"
	"strconv"
)

const (
	miB = 1 << 20
	giB = 1 << 30

	// minMemoryBudget is the floor for the memory budget: below this,
	// cloning anything non-trivial is not realistic anyway.
	minMemoryBudget = 256 * miB
	// defaultMemoryBudget is used when the available memory cannot be
	// determined.
	defaultMemoryBudget = 1 * giB
)

// availableMemory is overridable in tests; it reports the memory this
// process can still use (0 when unknown). Implemented per OS.
var availableMemory = systemAvailableMemory

// memoryBudget returns how much memory a single clone may use: half of
// what is currently available (respecting cgroup limits on Linux), so the
// rest of the system keeps breathing room.
func memoryBudget() uint64 {
	return budgetFrom(availableMemory())
}

func budgetFrom(available uint64) uint64 {
	if available == 0 {
		return defaultMemoryBudget
	}
	return max(available/2, minMemoryBudget)
}

// gitMemoryConfig translates a memory budget into git config settings
// that bound git's memory use while receiving and indexing a pack:
//
//   - pack.threads: index-pack resolves deltas with this many threads,
//     each with its own delta base cache.
//   - core.deltaBaseCacheLimit: the per-thread delta base cache.
//   - core.bigFileThreshold: blobs above this are streamed instead of
//     being held in memory as a whole.
func gitMemoryConfig(budget uint64) []string {
	threads := clamp(budget/(256*miB), 1, uint64(min(runtime.NumCPU(), 8)))
	deltaBaseCache := clamp(budget/(4*threads), 16*miB, 256*miB)
	bigFileThreshold := clamp(budget/8, 16*miB, 512*miB)

	return []string{
		"pack.threads=" + strconv.FormatUint(threads, 10),
		"core.deltaBaseCacheLimit=" + strconv.FormatUint(deltaBaseCache, 10),
		"core.bigFileThreshold=" + strconv.FormatUint(bigFileThreshold, 10),
	}
}

func clamp(v, lo, hi uint64) uint64 {
	return max(lo, min(v, hi))
}

func formatMiB(b uint64) string {
	return fmt.Sprintf("%d MiB", b/miB)
}
