package internal

import (
	"bytes"
	"context"
	"kakaotalkadblock/internal/beta"
	"kakaotalkadblock/internal/win/winapi"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const sleepTime = 100 * time.Millisecond
// 500ms combined with the 5s snapshot pass measured 0.4% idle CPU, so the interval stays at 1000ms.
const watchInterval = 1000 * time.Millisecond

// snapshotEvery bounds how long a second instance can go undetected while a pid is already tracked.
const snapshotEvery = 5 * time.Second
const executable = "kakaotalk.exe"

var executableBytes = []byte(executable)

var mutex = &sync.Mutex{}
var mainWindowHandleMap = make(map[windows.HWND]struct{})
var adSubwindowCandidateMap = make(map[windows.HWND]struct{})
var windowTextMap = make(map[windows.HWND]string)
var windowClassMap = make(map[windows.HWND]string)
var customScrollHandleMap = make(map[windows.HWND]bool)
var windowParentMap = make(map[windows.HWND]windows.HWND)

// mainWindowRecheck bounds how stale a cached isMainWindow result may get.
const mainWindowRecheck = time.Second

type mainWindowState struct {
	isMain    bool
	checkedAt time.Time
}

var mainWindowStateMap = make(map[windows.HWND]mainWindowState)

// lastSeen maps a target pid to any one of its windows, whatever the class.
var lastSeen = make(map[uint32]windows.HWND)

// childCollector is shared by every collectChildren call and is only touched
// while mutex is held.
var childCollector []windows.HWND
var childCollectorCallback = syscall.NewCallback(func(handle windows.HWND, _ uintptr) uintptr {
	childCollector = append(childCollector, handle)
	return 1
})

func watch(ctx context.Context) {
	var (
		pe32 winapi.ProcessEntry32
	)
	pe32.DwSize = uint32(unsafe.Sizeof(pe32))
	var snapshot windows.HWND
	// Declared outside the callback so &windowPid is not heap-allocated per window.
	var windowPid uint32
	var enumWindow = syscall.NewCallback(func(handle windows.HWND, processId uintptr) uintptr {
		winapi.GetWindowThreadProcessId(handle, &windowPid)
		if processId == uintptr(windowPid) {
			if _, ok := lastSeen[windowPid]; !ok {
				lastSeen[windowPid] = handle
			}
			className := winapi.GetClassName(handle)
			parentHandle := winapi.GetParent(handle)
			beta.TrackWindow(handle, className, parentHandle)
			if className == "EVA_Window_Dblclk" || className == "EVA_Window" {
				windowText := winapi.GetWindowText(handle)

				switch className {
				case "EVA_Window_Dblclk":
					if windowText != "" && parentHandle == 0 {
						mainWindowHandleMap[handle] = struct{}{}
					} else if windowText == "" && parentHandle != 0 {
						if _, ok := mainWindowHandleMap[parentHandle]; ok {
							adSubwindowCandidateMap[handle] = struct{}{}
						}
					}
				case "EVA_Window":
					if windowText == "" && parentHandle == 0 {
						adSubwindowCandidateMap[handle] = struct{}{}
					}
				}
			}
		}
		return 1
	})
	ticker := time.NewTicker(watchInterval)
	defer ticker.Stop()
	var lastSnapshot time.Time
	// Closure, not `defer winapi.CloseHandle(snapshot)`: deferred arguments are
	// evaluated immediately, which would capture the zero handle.
	defer func() { winapi.CloseHandle(snapshot) }()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			Mark("watch_tick")
			mutex.Lock()
			for pid, hwnd := range lastSeen {
				if !winapi.IsWindow(hwnd) {
					delete(lastSeen, pid)
				}
			}
			evictDeadWindows()
			if len(lastSeen) > 0 && time.Since(lastSnapshot) < snapshotEvery {
				for pid := range lastSeen {
					winapi.EnumWindows(enumWindow, uintptr(pid))
				}
				mutex.Unlock()
				continue
			}
			lastSnapshot = time.Now()
			// Release the previous snapshot before taking a new one,
			// otherwise every refresh leaks a kernel snapshot object.
			winapi.CloseHandle(snapshot)
			snapshot = winapi.CreateToolhelp32Snapshot(winapi.Th32csSnapprocess, 0)
			if winapi.Process32First(uintptr(snapshot), &pe32) {
				for {
					name := pe32.SzExeFile[:]
					if n := bytes.IndexByte(name, 0); n >= 0 {
						name = name[:n]
					}

					if bytes.EqualFold(name, executableBytes) || beta.IsExecutable(name) {
						pid := pe32.Th32ProcessID
						winapi.EnumWindows(enumWindow, uintptr(pid))
					}

					if !winapi.Process32Next(uintptr(snapshot), &pe32) {
						break
					}
				}
			}
			winapi.CloseHandle(snapshot)
			snapshot = 0
			mutex.Unlock()
		}
	}
}

// windowLiveness is reused by every evictDeadWindows pass so the pass does not allocate.
var windowLiveness = make(map[windows.HWND]bool)

// evictDeadWindows drops handles of destroyed windows from the per-window
// caches. The caller must hold mutex.
func evictDeadWindows() {
	clear(windowLiveness)
	for h := range mainWindowHandleMap {
		evictIfDead(h)
	}
	for h := range adSubwindowCandidateMap {
		evictIfDead(h)
	}
	for h := range windowTextMap {
		evictIfDead(h)
	}
	for h := range windowClassMap {
		evictIfDead(h)
	}
	for h := range customScrollHandleMap {
		evictIfDead(h)
	}
	for h := range mainWindowStateMap {
		evictIfDead(h)
	}
	for h := range windowParentMap {
		evictIfDead(h)
	}
}

func evictIfDead(h windows.HWND) {
	alive, checked := windowLiveness[h]
	if !checked {
		alive = winapi.IsWindow(h)
		windowLiveness[h] = alive
	}
	if alive {
		return
	}
	delete(mainWindowHandleMap, h)
	delete(adSubwindowCandidateMap, h)
	delete(windowTextMap, h)
	delete(windowClassMap, h)
	delete(customScrollHandleMap, h)
	delete(mainWindowStateMap, h)
	delete(windowParentMap, h)
}

func removeAd(ctx context.Context) {
	childHandles := make([]windows.HWND, 0)
	ticker := time.NewTicker(sleepTime)

	defer ticker.Stop()
	defer beta.Restore()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			Mark("removeAd_tick")
			mutex.Lock()
			now := time.Now()
			var rect winapi.Rect
			for wnd := range mainWindowHandleMap {
				if wnd == 0 {
					continue
				}
				state, known := mainWindowStateMap[wnd]
				fresh := known && now.Sub(state.checkedAt) < mainWindowRecheck
				if fresh && !state.isMain {
					continue
				}
				// Copy: classNameStartsWith below reuses the collector.
				childHandles = append(childHandles[:0], collectChildren(wnd)...)

				if !fresh {
					state = mainWindowState{isMain: isMainWindow(childHandles), checkedAt: now}
					mainWindowStateMap[wnd] = state
				}
				if !state.isMain || len(childHandles) == 0 {
					continue
				}

				rect = winapi.Rect{}
				winapi.GetWindowRect(wnd, &rect)
				for i, childHandle := range childHandles {
					className := getWindowClass(childHandle)
					windowText := getWindowText(childHandle)
					parentHandle := getWindowParent(childHandle)
					if parentHandle != wnd {
						continue
					}
					// The top-most child is never sent WM_CLOSE (video playback window).
					if i > 0 {
						parentText := getWindowText(parentHandle)
						if className == "EVA_ChildWindow" && windowText == "" && parentText != "" {
							hasCustomScroll, ok := customScrollHandleMap[wnd]
							if !ok {
								hasCustomScroll = classNameStartsWith(childHandle, "_EVA_")
								customScrollHandleMap[wnd] = hasCustomScroll
							}
							if !hasCustomScroll {
								winapi.SendMessage(childHandle, winapi.WmClose, 0, 0)
							}
						}
					}
					HideMainViewAdArea(windowText, &rect, childHandle)
					HideLockScreenAdArea(windowText, &rect, childHandle)
				}
			}
			for wnd := range adSubwindowCandidateMap {
				if !winapi.IsWindowVisible(wnd) {
					continue
				}
				if hasChromeLegacyWindow(wnd) {
					markPopupHide(wnd)
					winapi.ShowWindow(wnd, 0)
				}
			}
			beta.RemoveAds()
			mutex.Unlock()
		}
	}
}

// collectChildren returns every descendant of parent. The slice is backed by
// childCollector, so the next call overwrites it.
func collectChildren(parent windows.HWND) []windows.HWND {
	childCollector = childCollector[:0]
	winapi.EnumChildWindows(parent, childCollectorCallback, 0)
	return childCollector
}

func classNameStartsWith(handle windows.HWND, className string) bool {
	if strings.HasPrefix(getWindowClass(handle), className) {
		return true
	}
	for _, wnd := range collectChildren(handle) {
		if strings.HasPrefix(getWindowClass(wnd), className) {
			return true
		}
	}
	return false
}

func hasChromeLegacyWindow(handle windows.HWND) bool {
	if getWindowText(handle) == "Chrome Legacy Window" {
		return true
	}
	for _, wnd := range collectChildren(handle) {
		if getWindowText(wnd) == "Chrome Legacy Window" {
			return true
		}
	}
	return false
}

func isMainWindow(handles []windows.HWND) bool {
	for _, wnd := range handles {
		windowClass := getWindowClass(wnd)
		if windowClass != "EVA_ChildWindow" {
			continue
		}

		windowText := winapi.GetWindowText(wnd)
		if strings.HasPrefix(windowText, "OnlineMainView") || strings.HasPrefix(windowText, "LockModeView") {
			return true
		}
	}
	return false
}

func getWindowText(handle windows.HWND) string {
	text, ok := windowTextMap[handle]
	if !ok {
		text = winapi.GetWindowText(handle)
		windowTextMap[handle] = text
	}
	return text
}

func getWindowParent(handle windows.HWND) windows.HWND {
	parent, ok := windowParentMap[handle]
	if !ok {
		parent = winapi.GetParent(handle)
		windowParentMap[handle] = parent
	}
	return parent
}

func getWindowClass(handle windows.HWND) string {
	class, ok := windowClassMap[handle]
	if !ok {
		class = winapi.GetClassName(handle)
		windowClassMap[handle] = class
	}
	return class
}

func Run(ctx context.Context) {
	go watch(ctx)
	go removeAd(ctx)
}
