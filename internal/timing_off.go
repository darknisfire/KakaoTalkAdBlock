//go:build !timing

package internal

import "golang.org/x/sys/windows"

func Mark(string) {}

func markPopupHide(windows.HWND) {}
