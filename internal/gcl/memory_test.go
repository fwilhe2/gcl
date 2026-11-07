package gcl

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestBudgetFrom(t *testing.T) {
	tests := []struct{ available, want uint64 }{
		{0, defaultMemoryBudget},
		{100 * miB, minMemoryBudget},
		{8 * giB, 4 * giB},
	}
	for _, tt := range tests {
		if got := budgetFrom(tt.available); got != tt.want {
			t.Errorf("budgetFrom(%d) = %d, want %d", tt.available, got, tt.want)
		}
	}
}

func gitConfigValue(t *testing.T, settings []string, key string) uint64 {
	t.Helper()
	for _, s := range settings {
		if v, ok := strings.CutPrefix(s, key+"="); ok {
			n, err := strconv.ParseUint(v, 10, 64)
			if err != nil {
				t.Fatal(err)
			}
			return n
		}
	}
	t.Fatalf("%s missing in %q", key, settings)
	return 0
}

func TestGitMemoryConfigStaysWithinBudget(t *testing.T) {
	for _, budget := range []uint64{minMemoryBudget, 512 * miB, 2 * giB, 32 * giB} {
		settings := gitMemoryConfig(budget)
		threads := gitConfigValue(t, settings, "pack.threads")
		cache := gitConfigValue(t, settings, "core.deltaBaseCacheLimit")
		bigFile := gitConfigValue(t, settings, "core.bigFileThreshold")
		if threads < 1 {
			t.Errorf("budget %d: pack.threads = %d", budget, threads)
		}
		// index-pack holds one delta base cache per thread, plus up to
		// one big object being inflated.
		if used := threads*cache + bigFile; used > budget {
			t.Errorf("budget %s: caches may use %s", formatMiB(budget), formatMiB(used))
		}
	}
}

func TestGuardMemoryReportsExceededLimit(t *testing.T) {
	previous := memoryPollInterval
	memoryPollInterval = time.Millisecond
	t.Cleanup(func() { memoryPollInterval = previous })

	exceeded := make(chan uint64, 1)
	stop := guardMemory(1, func(used uint64) { exceeded <- used })
	defer stop()

	select {
	case used := <-exceeded:
		if used <= 1 {
			t.Fatalf("reported %d bytes", used)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("guard did not trigger")
	}
}

func TestGuardMemoryStaysQuietBelowLimit(t *testing.T) {
	previous := memoryPollInterval
	memoryPollInterval = time.Millisecond
	t.Cleanup(func() { memoryPollInterval = previous })

	stop := guardMemory(1<<50, func(uint64) { t.Error("guard triggered below the limit") })
	time.Sleep(20 * time.Millisecond)
	stop()
}

func TestAbortOnMemoryLimitCleansUpAndExits(t *testing.T) {
	previous := exitProcess
	t.Cleanup(func() { exitProcess = previous })
	var code int
	exitProcess = func(c int) { code = c }

	cleaned := false
	abortOnMemoryLimit("clone of x", minMemoryBudget, func() { cleaned = true })(2 * minMemoryBudget)
	if !cleaned || code != 1 {
		t.Fatalf("cleaned = %v, exit code = %d", cleaned, code)
	}
}
