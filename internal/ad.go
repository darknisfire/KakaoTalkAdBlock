package internal

import (
	"strings"

	"golang.org/x/sys/windows"

	"kakaotalkadblock/internal/win/winapi"
)

const (
	LayoutShadowPadding = 2
	MainViewPadding     = 31
)

// sizeMatches skips redundant SetWindowPos calls, which would also re-raise the view in z-order.
func sizeMatches(handle windows.HWND, width, height int32) bool {
	var cur winapi.Rect
	if !winapi.GetWindowRect(handle, &cur) {
		return false
	}
	return cur.Right-cur.Left == width && cur.Bottom-cur.Top == height
}

func HideLockScreenAdArea(windowText string, rect *winapi.Rect, handle windows.HWND) {
	if strings.HasPrefix(windowText, "LockModeView") {
		width := rect.Right - rect.Left - LayoutShadowPadding
		height := rect.Bottom - rect.Top
		if sizeMatches(handle, width, height) {
			return
		}
		winapi.UpdateWindow(handle)
		winapi.SetWindowPos(handle, 0, 0, 0, width, height, winapi.SwpNomove|winapi.SwpNozorder)
	}
}

func HideMainViewAdArea(windowText string, rect *winapi.Rect, handle windows.HWND) {
	if strings.HasPrefix(windowText, "OnlineMainView") {
		width := rect.Right - rect.Left - LayoutShadowPadding
		height := rect.Bottom - rect.Top - MainViewPadding
		if height < 1 {
			return
		}
		if sizeMatches(handle, width, height) {
			return
		}
		winapi.UpdateWindow(handle)
		Mark("first_hide_mainview")
		winapi.SetWindowPos(handle, 0, 0, 0, width, height, winapi.SwpNomove|winapi.SwpNozorder)
	}
}
