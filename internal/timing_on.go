//go:build timing

package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/metrics"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/sys/windows"

	"kakaotalkadblock/internal/win/winapi"
)

const reportInterval = 10 * time.Second

var (
	startTime = processStartTime()
	logMu     sync.Mutex
	logFile   *os.File
	marked    sync.Map
)

// processStartTime falls back to package init time when the OS call fails.
func processStartTime() time.Time {
	var creation, exit, kernel, user windows.Filetime
	if err := windows.GetProcessTimes(windows.CurrentProcess(), &creation, &exit, &kernel, &user); err != nil {
		return time.Now()
	}
	return time.Unix(0, creation.Nanoseconds())
}

func elapsedMs() int64 {
	return time.Since(startTime).Milliseconds()
}

func writeLog(format string, args ...any) {
	logMu.Lock()
	defer logMu.Unlock()
	if logFile == nil {
		return
	}
	fmt.Fprintf(logFile, format+"\n", args...)
}

// Mark records the first call for each name; later calls are ignored.
func Mark(name string) {
	if _, dup := marked.LoadOrStore(name, struct{}{}); dup {
		return
	}
	writeLog("mark name=%s t_ms=%d", name, elapsedMs())
}

func markPopupHide(hwnd windows.HWND) {
	writeLog("event popup_hide hwnd=0x%x t_ms=%d", uintptr(hwnd), elapsedMs())
}

func mapSizes() (mainWindow, adCandidate, windowText, windowClass, customScroll, seen, mainState, windowParent int) {
	mutex.Lock()
	defer mutex.Unlock()
	return len(mainWindowHandleMap), len(adSubwindowCandidateMap), len(windowTextMap),
		len(windowClassMap), len(customScrollHandleMap), len(lastSeen),
		len(mainWindowStateMap), len(windowParentMap)
}

func readAllocs() uint64 {
	s := []metrics.Sample{{Name: "/gc/heap/allocs:objects"}}
	metrics.Read(s)
	if s[0].Value.Kind() != metrics.KindUint64 {
		return 0
	}
	return s[0].Value.Uint64()
}

func report(prev map[string]uint64, prevAllocs uint64) (map[string]uint64, uint64) {
	cur := winapi.Snapshot()
	allocs := readAllocs()
	names := make([]string, 0, len(cur))
	for k := range cur {
		names = append(names, k)
	}
	sort.Strings(names)

	secs := reportInterval.Seconds()
	var b strings.Builder
	fmt.Fprintf(&b, "report t_ms=%d", elapsedMs())
	for _, k := range names {
		fmt.Fprintf(&b, " %s_ps=%.1f", k, float64(cur[k]-prev[k])/secs)
	}
	mw, ac, wt, wc, cs, ls, ms, wp := mapSizes()
	fmt.Fprintf(&b, " allocs_ps=%.1f goroutines=%d map_mainWindow=%d map_adCandidate=%d map_windowText=%d map_windowClass=%d map_customScroll=%d map_lastSeen=%d map_mainState=%d map_windowParent=%d",
		float64(allocs-prevAllocs)/secs, runtime.NumGoroutine(), mw, ac, wt, wc, cs, ls, ms, wp)
	var mst runtime.MemStats
	runtime.ReadMemStats(&mst)
	fmt.Fprintf(&b, " heap_alloc_kb=%d heap_inuse_kb=%d heap_sys_kb=%d heap_released_kb=%d stack_sys_kb=%d mspan_sys_kb=%d mcache_sys_kb=%d gc_sys_kb=%d other_sys_kb=%d sys_total_kb=%d num_gc=%d",
		mst.HeapAlloc>>10, mst.HeapInuse>>10, mst.HeapSys>>10, mst.HeapReleased>>10, mst.StackSys>>10, mst.MSpanSys>>10, mst.MCacheSys>>10, mst.GCSys>>10, mst.OtherSys>>10, mst.Sys>>10, mst.NumGC)
	writeLog("%s", b.String())
	return cur, allocs
}

func init() {
	if testing.Testing() {
		return
	}
	path := filepath.Join(os.TempDir(), "ktab-timing.log")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	logFile = f
	writeLog("--- start pid=%d ---", os.Getpid())

	go func() {
		prev := winapi.Snapshot()
		prevAllocs := readAllocs()
		ticker := time.NewTicker(reportInterval)
		defer ticker.Stop()
		for range ticker.C {
			prev, prevAllocs = report(prev, prevAllocs)
		}
	}()
}
