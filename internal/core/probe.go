package core

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/metrics"
	"runtime/pprof"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sys/unix"
)

// Throwaway contention probe: named event counters, runtime metrics, and
// mutex/block profiles, all gated by TSGO_PROBE. Report is printed at exit.

var ProbeEnabled = os.Getenv("TSGO_PROBE") != ""

type Counter struct {
	name string
	v    atomic.Uint64
}

var (
	probeMu       sync.Mutex
	probeCounters []*Counter
	probeStart    time.Time
)

func NewCounter(name string) *Counter {
	c := &Counter{name: name}
	probeMu.Lock()
	probeCounters = append(probeCounters, c)
	probeMu.Unlock()
	return c
}

func (c *Counter) Inc() {
	if ProbeEnabled {
		c.v.Add(1)
	}
}

func (c *Counter) Add(n uint64) {
	if ProbeEnabled {
		c.v.Add(n)
	}
}

func (c *Counter) Load() uint64 {
	return c.v.Load()
}

var (
	probeLinesMu sync.Mutex
	probeLines   []string
)

// ProbeLine appends a preformatted line to the exit report.
func ProbeLine(format string, args ...any) {
	if !ProbeEnabled {
		return
	}
	probeLinesMu.Lock()
	probeLines = append(probeLines, fmt.Sprintf(format, args...))
	probeLinesMu.Unlock()
}

func StartProbe() {
	if !ProbeEnabled {
		return
	}
	probeStart = time.Now()
	if os.Getenv("TSGO_PROBE_LATE") == "" {
		runtime.SetMutexProfileFraction(1)
		runtime.SetBlockProfileRate(10000)
	}
}

// ProbeMark records a snapshot of a few runtime metrics at a phase boundary.
func ProbeMark(name string) {
	if !ProbeEnabled {
		return
	}
	if name == "beforeCheckGroup" && os.Getenv("TSGO_PROBE_LATE") != "" {
		runtime.SetMutexProfileFraction(1)
		runtime.SetBlockProfileRate(10000)
	}
	names := []string{
		"/sync/mutex/wait/total:seconds",
		"/cpu/classes/user:cpu-seconds",
		"/cpu/classes/gc/total:cpu-seconds",
		"/gc/cycles/total:gc-cycles",
		"/gc/heap/allocs:bytes",
		"/sched/goroutines:goroutines",
	}
	samples := make([]metrics.Sample, len(names))
	for i, n := range names {
		samples[i].Name = n
	}
	metrics.Read(samples)
	var ru unix.Rusage
	_ = unix.Getrusage(unix.RUSAGE_SELF, &ru)
	ProbeLine("mark t=%7.3fs %-16s mutexwait=%.1fs usercpu=%.1fs gccpu=%.1fs gcs=%d allocGB=%.2f goroutines=%d ruser=%.2fs rsys=%.2fs",
		time.Since(probeStart).Seconds(), name,
		samples[0].Value.Float64(), samples[1].Value.Float64(), samples[2].Value.Float64(),
		samples[3].Value.Uint64(), float64(samples[4].Value.Uint64())/1e9, samples[5].Value.Uint64(),
		float64(ru.Utime.Nano())/1e9, float64(ru.Stime.Nano())/1e9)
}

func StopProbe() {
	if !ProbeEnabled {
		return
	}
	elapsed := time.Since(probeStart)
	w := os.Stderr
	var sb strings.Builder
	fmt.Fprintf(&sb, "probe: elapsed=%.3fs gomaxprocs=%d numcpu=%d numgc=%d\n", elapsed.Seconds(), runtime.GOMAXPROCS(0), runtime.NumCPU(), numGC())
	writeRuntimeMetrics(&sb)
	probeLinesMu.Lock()
	lines := probeLines
	probeLinesMu.Unlock()
	sort.Strings(lines)
	for _, line := range lines {
		fmt.Fprintf(&sb, "probe: %s\n", line)
	}
	probeMu.Lock()
	for _, c := range probeCounters {
		fmt.Fprintf(&sb, "probe: counter %-40s %d\n", c.name, c.v.Load())
	}
	probeMu.Unlock()
	fmt.Fprint(w, sb.String())
	if dir := os.Getenv("TSGO_PROBE"); dir != "" && dir != "1" {
		writeProfile(dir, "mutex")
		writeProfile(dir, "block")
		writeProfile(dir, "goroutine")
	}
}

func numGC() uint32 {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return ms.NumGC
}

func writeProfile(dir string, name string) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	f, err := os.Create(filepath.Join(dir, fmt.Sprintf("%d-%s.pb.gz", os.Getpid(), name)))
	if err != nil {
		return
	}
	defer f.Close()
	if p := pprof.Lookup(name); p != nil {
		_ = p.WriteTo(f, 0)
	}
}

func writeRuntimeMetrics(sb *strings.Builder) {
	names := []string{
		"/cpu/classes/total:cpu-seconds",
		"/cpu/classes/user:cpu-seconds",
		"/cpu/classes/idle:cpu-seconds",
		"/cpu/classes/gc/total:cpu-seconds",
		"/cpu/classes/gc/mark/assist:cpu-seconds",
		"/cpu/classes/gc/mark/dedicated:cpu-seconds",
		"/cpu/classes/gc/mark/idle:cpu-seconds",
		"/cpu/classes/gc/pause:cpu-seconds",
		"/cpu/classes/scavenge/total:cpu-seconds",
		"/gc/cycles/total:gc-cycles",
		"/gc/heap/allocs:bytes",
		"/gc/heap/allocs:objects",
		"/gc/heap/goal:bytes",
		"/gc/pauses:seconds",
		"/sched/latencies:seconds",
		"/sched/gomaxprocs:threads",
		"/sched/goroutines:goroutines",
		"/sync/mutex/wait/total:seconds",
		"/memory/classes/total:bytes",
		"/memory/classes/heap/objects:bytes",
	}
	samples := make([]metrics.Sample, len(names))
	for i, n := range names {
		samples[i].Name = n
	}
	metrics.Read(samples)
	for _, s := range samples {
		switch s.Value.Kind() {
		case metrics.KindUint64:
			fmt.Fprintf(sb, "probe: metric %-45s %d\n", s.Name, s.Value.Uint64())
		case metrics.KindFloat64:
			fmt.Fprintf(sb, "probe: metric %-45s %.4f\n", s.Name, s.Value.Float64())
		case metrics.KindFloat64Histogram:
			h := s.Value.Float64Histogram()
			var total uint64
			for _, c := range h.Counts {
				total += c
			}
			fmt.Fprintf(sb, "probe: metric %-45s count=%d p50=%s p99=%s p999=%s max=%s\n", s.Name, total,
				fmtSeconds(histQuantile(h, 0.50)), fmtSeconds(histQuantile(h, 0.99)), fmtSeconds(histQuantile(h, 0.999)), fmtSeconds(histQuantile(h, 1.0)))
		case metrics.KindBad:
			fmt.Fprintf(sb, "probe: metric %-45s (unsupported)\n", s.Name)
		}
	}
}

func histQuantile(h *metrics.Float64Histogram, q float64) float64 {
	var total uint64
	for _, c := range h.Counts {
		total += c
	}
	if total == 0 {
		return 0
	}
	target := uint64(float64(total) * q)
	if target >= total {
		target = total - 1
	}
	var cum uint64
	for i, c := range h.Counts {
		cum += c
		if cum > target {
			return h.Buckets[i+1]
		}
	}
	return h.Buckets[len(h.Buckets)-1]
}

func fmtSeconds(s float64) string {
	switch {
	case s >= 1:
		return fmt.Sprintf("%.2fs", s)
	case s >= 1e-3:
		return fmt.Sprintf("%.2fms", s*1e3)
	case s >= 1e-6:
		return fmt.Sprintf("%.2fus", s*1e6)
	default:
		return fmt.Sprintf("%.0fns", s*1e9)
	}
}
