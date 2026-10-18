package winapi

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

type Rect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

var (
	user32                       = windows.NewLazySystemDLL("user32.dll")
	getClassName                 = user32.NewProc("GetClassNameW")
	enumChildWindows             = user32.NewProc("EnumChildWindows")
	enumWindows                  = user32.NewProc("EnumWindows")
	showWindow                   = user32.NewProc("ShowWindow")
	getParent                    = user32.NewProc("GetParent")
	setWindowPos                 = user32.NewProc("SetWindowPos")
	getWindowText                = user32.NewProc("GetWindowTextW")
	getWindowRect                = user32.NewProc("GetWindowRect")
	getDpiForWindow              = user32.NewProc("GetDpiForWindow")
	isWindow                     = user32.NewProc("IsWindow")
	isWindowVisible              = user32.NewProc("IsWindowVisible")
	isZoomed                     = user32.NewProc("IsZoomed")
	updateWindow                 = user32.NewProc("UpdateWindow")
	sendMessage                  = user32.NewProc("SendMessageW")
	getWindowThreadProcessId     = user32.NewProc("GetWindowThreadProcessId")
	moveWindow                   = user32.NewProc("MoveWindow")
	setWindowRgn                 = user32.NewProc("SetWindowRgn")
	redrawWindow                 = user32.NewProc("RedrawWindow")
	setThreadDpiAwarenessContext = user32.NewProc("SetThreadDpiAwarenessContext")
)

var windowStyleIndexArg = uintptr(^uint32(15))

// SyscallN keeps its args on the stack; LazyProc.Call (//go:uintptrescapes) heap-allocates them and any pointed-to buffer.
func GetClassName(hWnd windows.HWND) string {
	count(cGetClassName)
	var buf [255]uint16
	syscall.SyscallN(getClassName.Addr(), uintptr(hWnd), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf[:])
}

func EnumWindows(lpEnumFunc, lParam uintptr) bool {
	count(cEnumWindows)
	r, _, _ := syscall.SyscallN(enumWindows.Addr(), lpEnumFunc, lParam)
	return r != 0
}
func EnumChildWindows(hWndParent windows.HWND, lpEnumFunc, lParam uintptr) bool {
	count(cEnumChildWindows)
	r, _, _ := syscall.SyscallN(enumChildWindows.Addr(), uintptr(hWndParent), lpEnumFunc, lParam)
	return r != 0
}

func ShowWindow(hWnd windows.HWND, nCmdShow int32) bool {
	count(cShowWindow)
	r, _, _ := syscall.SyscallN(showWindow.Addr(), uintptr(hWnd), uintptr(nCmdShow))
	return r != 0
}

func GetParent(hWnd windows.HWND) windows.HWND {
	count(cGetParent)
	r, _, _ := syscall.SyscallN(getParent.Addr(), uintptr(hWnd))
	return windows.HWND(r)
}

func SetWindowPos(hWnd, hWndInsertAfter windows.HWND, x, y, cx, cy int32, uFlags uint32) bool {
	count(cSetWindowPos)
	r, _, _ := syscall.SyscallN(setWindowPos.Addr(), uintptr(hWnd), uintptr(hWndInsertAfter), uintptr(x), uintptr(y), uintptr(cx), uintptr(cy), uintptr(uFlags))
	return r != 0
}

func GetWindowText(hWnd windows.HWND) string {
	count(cGetWindowText)
	var buf [255]uint16
	syscall.SyscallN(getWindowText.Addr(), uintptr(hWnd), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf[:])
}

func GetWindowRect(hWnd windows.HWND, lpRect *Rect) bool {
	count(cGetWindowRect)
	r, _, _ := syscall.SyscallN(getWindowRect.Addr(), uintptr(hWnd), uintptr(unsafe.Pointer(lpRect)))
	return r != 0
}

func GetWindowStyle(hWnd windows.HWND) uintptr {
	style, _, _ := syscall.SyscallN(getWindowLongPtr.Addr(), uintptr(hWnd), windowStyleIndexArg)
	return style
}

func SetWindowStyle(hWnd windows.HWND, style uintptr) bool {
	syscall.SyscallN(setWindowLongPtr.Addr(), uintptr(hWnd), windowStyleIndexArg, style)
	return GetWindowStyle(hWnd) == style
}

func IsWindow(hWnd windows.HWND) bool {
	count(cIsWindow)
	result, _, _ := syscall.SyscallN(isWindow.Addr(), uintptr(hWnd))
	return result != 0
}

func IsWindowVisible(hWnd windows.HWND) bool {
	count(cIsWindowVisible)
	result, _, _ := syscall.SyscallN(isWindowVisible.Addr(), uintptr(hWnd))
	return result != 0
}

func IsZoomed(hWnd windows.HWND) bool {
	result, _, _ := syscall.SyscallN(isZoomed.Addr(), uintptr(hWnd))
	return result != 0
}

func UpdateWindow(hWnd windows.HWND) bool {
	count(cUpdateWindow)
	ret, _, _ := syscall.SyscallN(updateWindow.Addr(), uintptr(hWnd))
	return ret != 0
}

func SendMessage(hWd windows.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	count(cSendMessage)
	r, _, _ := syscall.SyscallN(sendMessage.Addr(), uintptr(hWd), uintptr(msg), wParam, lParam)
	return r
}

func GetWindowThreadProcessId(hWnd windows.HWND, dwProcessId *uint32) uint32 {
	count(cGetWindowThreadProcessId)
	r, _, _ := syscall.SyscallN(getWindowThreadProcessId.Addr(), uintptr(hWnd), uintptr(unsafe.Pointer(dwProcessId)))
	return uint32(r)
}

func MoveWindow(hWnd windows.HWND, x, y, width, height int32, repaint bool) bool {
	shouldRepaint := 0
	if repaint {
		shouldRepaint = 1
	}
	r, _, _ := syscall.SyscallN(
		moveWindow.Addr(),
		uintptr(hWnd),
		uintptr(x),
		uintptr(y),
		uintptr(width),
		uintptr(height),
		uintptr(shouldRepaint),
	)
	return r != 0
}

func GetDpiForWindow(hWnd windows.HWND) uint32 {
	if err := getDpiForWindow.Find(); err != nil {
		return 0
	}
	dpi, _, _ := syscall.SyscallN(getDpiForWindow.Addr(), uintptr(hWnd))
	return uint32(dpi)
}

func SetWindowRgn(hWnd windows.HWND, region uintptr, redraw bool) bool {
	shouldRedraw := 0
	if redraw {
		shouldRedraw = 1
	}
	r, _, _ := syscall.SyscallN(setWindowRgn.Addr(), uintptr(hWnd), region, uintptr(shouldRedraw))
	return r != 0
}

func RefreshWindowFrame(hWnd windows.HWND) bool {
	flags := uint32(SwpNosize | SwpNomove | SwpNozorder | SwpNoactivate | SwpFramechanged)
	return SetWindowPos(hWnd, 0, 0, 0, 0, 0, flags)
}

func RedrawWindow(hWnd windows.HWND) bool {
	const flags = 0x0001 | 0x0080 | 0x0100 | 0x0400
	result, _, _ := syscall.SyscallN(redrawWindow.Addr(), uintptr(hWnd), 0, 0, flags)
	return result != 0
}

func EnableThreadPerMonitorV2DpiAwareness() uintptr {
	if err := setThreadDpiAwarenessContext.Find(); err != nil {
		return 0
	}
	count(cSetThreadDpiAwarenessContext)
	previousContext, _, _ := syscall.SyscallN(setThreadDpiAwarenessContext.Addr(), ^uintptr(3))
	return previousContext
}

func RestoreThreadDpiAwareness(previousContext uintptr) {
	if previousContext != 0 {
		count(cSetThreadDpiAwarenessContext)
		syscall.SyscallN(setThreadDpiAwarenessContext.Addr(), previousContext)
	}
}
