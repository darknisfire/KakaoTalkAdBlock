//go:build timing

package winapi

import "sync/atomic"

type counter uint8

const (
	cEnumWindows counter = iota
	cEnumChildWindows
	cGetWindowThreadProcessId
	cGetWindowText
	cGetClassName
	cGetParent
	cIsWindow
	cIsWindowVisible
	cGetWindowRect
	cSetWindowPos
	cUpdateWindow
	cShowWindow
	cSendMessage
	cProcess32Next
	cSetThreadDpiAwarenessContext
	counterLen
)

var counterNames = [counterLen]string{
	"EnumWindows",
	"EnumChildWindows",
	"GetWindowThreadProcessId",
	"GetWindowText",
	"GetClassName",
	"GetParent",
	"IsWindow",
	"IsWindowVisible",
	"GetWindowRect",
	"SetWindowPos",
	"UpdateWindow",
	"ShowWindow",
	"SendMessage",
	"Process32Next",
	"SetThreadDpiAwarenessContext",
}

var counters [counterLen]atomic.Uint64

func count(c counter) { counters[c].Add(1) }

// Snapshot returns cumulative call counts keyed by API name.
func Snapshot() map[string]uint64 {
	m := make(map[string]uint64, counterLen)
	for i := range counters {
		m[counterNames[i]] = counters[i].Load()
	}
	return m
}
