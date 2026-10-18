//go:build windows

package winapi

import (
	"syscall"
	"testing"

	"golang.org/x/sys/windows"
)

func desktopWindow(t *testing.T) windows.HWND {
	t.Helper()
	proc := windows.NewLazySystemDLL("user32.dll").NewProc("GetDesktopWindow")
	h, _, _ := syscall.SyscallN(proc.Addr())
	if h == 0 {
		t.Fatal("GetDesktopWindow returned 0")
	}
	return windows.HWND(h)
}

func TestAllocs(t *testing.T) {
	desktop := desktopWindow(t)
	var r Rect
	cases := []struct {
		name string
		max  float64
		fn   func()
	}{
		{"GetClassName", 1, func() { GetClassName(desktop) }},
		{"GetWindowText", 1, func() { GetWindowText(desktop) }},
		{"GetParent", 0, func() { GetParent(desktop) }},
		{"IsWindow", 0, func() { IsWindow(desktop) }},
		{"GetWindowRect", 0, func() { GetWindowRect(desktop, &r) }},
	}
	for _, c := range cases {
		got := testing.AllocsPerRun(1000, c.fn)
		t.Logf("%s: %v allocs/op", c.name, got)
		if got > c.max {
			t.Errorf("%s: %v allocs/op, want <= %v", c.name, got, c.max)
		}
	}
}

func TestGetClassNameDesktop(t *testing.T) {
	if got := GetClassName(desktopWindow(t)); got != "#32769" {
		t.Errorf("GetClassName(desktop) = %q, want %q", got, "#32769")
	}
}
