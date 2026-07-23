package core

import (
	"fmt"
	"runtime"
	"time"

	"golang.org/x/sys/unix"
)

// TaskProbe measures wall time and the calling thread's CPU time across a task.
// It locks the goroutine to its OS thread so RUSAGE_THREAD reflects only this task.
type TaskProbe struct {
	name  string
	start time.Time
	user0 time.Duration
	sys0  time.Duration
	on    bool
}

func threadCPU() (user time.Duration, sys time.Duration) {
	var ru unix.Rusage
	if err := unix.Getrusage(unix.RUSAGE_THREAD, &ru); err != nil {
		return 0, 0
	}
	return time.Duration(ru.Utime.Nano()) * time.Nanosecond, time.Duration(ru.Stime.Nano()) * time.Nanosecond
}

func StartTaskProbe(name string) *TaskProbe {
	if !ProbeEnabled {
		return nil
	}
	runtime.LockOSThread()
	u, sy := threadCPU()
	return &TaskProbe{name: name, start: time.Now(), user0: u, sys0: sy, on: true}
}

func (t *TaskProbe) Stop(extraFormat string, args ...any) {
	if t == nil || !t.on {
		return
	}
	wall := time.Since(t.start)
	u, sy := threadCPU()
	user := u - t.user0
	sys := sy - t.sys0
	runtime.UnlockOSThread()
	extra := ""
	if extraFormat != "" {
		extra = " " + fmt.Sprintf(extraFormat, args...)
	}
	ProbeLine("task %-14s wall=%.3fs user=%.3fs sys=%.3fs%s", t.name, wall.Seconds(), user.Seconds(), sys.Seconds(), extra)
}
