package gcl

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime/debug"
	"runtime/metrics"
	"time"
)

var (
	exitProcess          = os.Exit
	memoryPollInterval   = 200 * time.Millisecond
	progressPollInterval = 2 * time.Second
)

// processMemory returns the memory the Go runtime holds from the OS,
// excluding what it has already returned: close to the process's RSS.
func processMemory() uint64 {
	samples := []metrics.Sample{
		{Name: "/memory/classes/total:bytes"},
		{Name: "/memory/classes/heap/released:bytes"},
	}
	metrics.Read(samples)
	total := samples[0].Value.Uint64()
	released := samples[1].Value.Uint64()
	if released > total {
		return 0
	}
	return total - released
}

// guardMemory keeps an in-process go-git operation within limit. go-git
// holds every resolved delta of a pack in memory until the whole pack is
// indexed and offers no way to cancel that, so the only safe reaction to
// exceeding the limit is to stop the process before the system runs out
// of memory. The GC is tuned to the limit first so that garbage never
// pushes the process over it. The returned function stops the guard.
func guardMemory(limit uint64, onExceeded func(used uint64)) (stop func()) {
	previous := debug.SetMemoryLimit(int64(limit / 4 * 3))
	done := make(chan struct{})
	finished := make(chan struct{})

	go func() {
		defer close(finished)
		ticker := time.NewTicker(memoryPollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if used := processMemory(); used > limit {
					onExceeded(used)
					return
				}
			}
		}
	}()

	return func() {
		close(done)
		<-finished
		debug.SetMemoryLimit(previous)
	}
}

// abortOnMemoryLimit is the guardMemory callback for the go-git fallback:
// it reports the problem, runs cleanup and exits.
func abortOnMemoryLimit(what string, limit uint64, cleanup func()) func(uint64) {
	return func(used uint64) {
		fmt.Fprintf(os.Stderr,
			"gcl: aborting %s: memory use reached %s, the limit is %s (half of the available memory).\n"+
				"gcl: install git to clone large repositories with bounded memory use.\n",
			what, formatMiB(used), formatMiB(limit))
		if cleanup != nil {
			cleanup()
		}
		exitProcess(1)
	}
}

// reportProgress prints how much has been written below dir every few
// seconds while it grows. go-git reports the remote's progress messages
// but nothing while receiving the pack and resolving deltas, which for a
// large repository can take minutes. The returned function stops it.
func reportProgress(dir string, w io.Writer) (stop func()) {
	done := make(chan struct{})
	finished := make(chan struct{})

	go func() {
		defer close(finished)
		ticker := time.NewTicker(progressPollInterval)
		defer ticker.Stop()
		start := time.Now()
		var last uint64
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				size := dirSize(dir)
				if size == last {
					continue
				}
				last = size
				rate := float64(size) / miB / time.Since(start).Seconds()
				fmt.Fprintf(w, "Receiving objects: %s (%.1f MiB/s)\n", formatMiB(size), rate)
			}
		}
	}()

	return func() {
		close(done)
		<-finished
	}
}

func dirSize(dir string) uint64 {
	var size uint64
	_ = filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.Type().IsRegular() {
			if info, err := d.Info(); err == nil {
				size += uint64(info.Size())
			}
		}
		return nil
	})
	return size
}
