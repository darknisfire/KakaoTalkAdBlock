//go:build windows

package internal

import (
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func clearWindowMaps() {
	mutex.Lock()
	defer mutex.Unlock()
	clear(mainWindowHandleMap)
	clear(adSubwindowCandidateMap)
	clear(windowTextMap)
	clear(windowClassMap)
	clear(customScrollHandleMap)
	clear(mainWindowStateMap)
	clear(windowParentMap)
}

func desktopWindow(t *testing.T) windows.HWND {
	t.Helper()
	r, _, _ := windows.NewLazySystemDLL("user32.dll").NewProc("GetDesktopWindow").Call()
	if r == 0 {
		t.Fatal("GetDesktopWindow returned 0")
	}
	return windows.HWND(r)
}

func TestEvictDeadWindows(t *testing.T) {
	mutex.Lock()
	defer mutex.Unlock()
	defer func() {
		clear(mainWindowHandleMap)
		clear(adSubwindowCandidateMap)
		clear(windowTextMap)
		clear(windowClassMap)
		clear(customScrollHandleMap)
		clear(mainWindowStateMap)
		clear(windowParentMap)
	}()

	desktop := desktopWindow(t)
	dead := []windows.HWND{0x7fff0001, 0x7fff0002, 0x7fff0003, 0x7fff0004, 0x7fff0005, 0x7fff0006, 0x7fff0007}

	mainWindowHandleMap[dead[0]] = struct{}{}
	adSubwindowCandidateMap[dead[1]] = struct{}{}
	windowTextMap[dead[2]] = "x"
	windowClassMap[dead[3]] = "x"
	customScrollHandleMap[dead[4]] = true
	mainWindowStateMap[dead[5]] = mainWindowState{isMain: true, checkedAt: time.Now()}
	windowParentMap[dead[6]] = desktop
	windowTextMap[desktop] = "desktop"
	// One dead handle shared by two maps must be removed from both.
	windowClassMap[dead[2]] = "x"

	evictDeadWindows()

	sizes := map[string]int{
		"mainWindowHandleMap":     len(mainWindowHandleMap),
		"adSubwindowCandidateMap": len(adSubwindowCandidateMap),
		"windowClassMap":          len(windowClassMap),
		"customScrollHandleMap":   len(customScrollHandleMap),
		"mainWindowStateMap":      len(mainWindowStateMap),
		"windowParentMap":         len(windowParentMap),
	}
	for name, n := range sizes {
		if n != 0 {
			t.Errorf("%s has %d entries, want 0", name, n)
		}
	}
	if len(windowTextMap) != 1 {
		t.Errorf("windowTextMap has %d entries, want 1", len(windowTextMap))
	}
	if _, ok := windowTextMap[desktop]; !ok {
		t.Error("live desktop key was evicted")
	}
}

func TestEvictDeadWindowsAllocs(t *testing.T) {
	clearWindowMaps()
	defer clearWindowMaps()
	desktop := desktopWindow(t)

	mutex.Lock()
	defer mutex.Unlock()
	windowTextMap[desktop] = "desktop"
	windowClassMap[desktop] = "desktop"
	windowParentMap[0x7fff0001] = desktop
	evictDeadWindows()
	windowParentMap[0x7fff0001] = desktop

	allocs := testing.AllocsPerRun(100, func() {
		evictDeadWindows()
		windowParentMap[0x7fff0001] = desktop
	})
	if allocs != 0 {
		t.Errorf("evictDeadWindows allocated %v times per run, want 0", allocs)
	}
}
