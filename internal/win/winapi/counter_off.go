//go:build !timing

package winapi

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
)

func count(counter) {}
