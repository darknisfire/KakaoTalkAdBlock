//go:build windows

package main

import (
	"context"
	"kakaotalkadblock/internal"
	"kakaotalkadblock/internal/win"
	"kakaotalkadblock/winres"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/energye/systray"
	"golang.org/x/sys/windows"

	_ "kakaotalkadblock/winres"
)

const VERSION = "2.2.3"

var singleInstanceMutex windows.Handle

func main() {
	internal.Mark("main")
	if !acquireSingleInstance() {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	destroy := func() {
		cancel()
		systray.Quit()
	}
	systray.SetOnRClick(func(menu systray.IMenu) {
		menu.ShowMenu()
	})
	systray.Run(func() {
		internal.Mark("onReady")
		systray.SetIcon(winres.IconData)
		systray.SetTooltip("KakaoTalkAdBlock")
		systray.AddMenuItem(VERSION, VERSION).Disable()
		checkRelease := systray.AddMenuItem("", "Check latest releases")
		checkRelease.Click(func() {
			exec.Command(
				filepath.Join(internal.SystemDir(), "rundll32.exe"),
				"url.dll,FileProtocolHandler",
				"https://github.com/darknisfire/KakaoTalkAdBlock/releases",
			).Start()
		})
		checkRelease.Hide()
		systray.AddSeparator()
		go func() {
			time.Sleep(15 * time.Second)
			select {
			case <-ctx.Done():
				return
			default:
			}
			tagName, hasNewRelease := internal.CheckLatestVersion(ctx, VERSION)
			internal.Mark("version_check_done")
			if hasNewRelease {
				checkRelease.SetTitle("New version available: " + tagName)
				checkRelease.Show()
			}
		}()

		startupItem := systray.AddMenuItem("Run on startup", "Run on startup")
		if win.IsStartupEnabled() {
			startupItem.Check()
		}
		startupItem.Click(func() {
			if startupItem.Checked() {
				startupItem.Uncheck()
				win.SetStartupEnabled(false)
			} else {
				startupItem.Check()
				win.SetStartupEnabled(true)
			}
		})
		systray.AddMenuItem("E&xit", "Exit").Click(destroy)

		internal.Run(ctx)
	}, func() {
		cancel()
	})
}

// Local\ scopes the mutex to the current session, so one instance per logged-in user.
func acquireSingleInstance() bool {
	name, _ := windows.UTF16PtrFromString(`Local\KakaoTalkAdBlock`)
	h, err := windows.CreateMutex(nil, false, name)
	if err == windows.ERROR_ALREADY_EXISTS {
		if h != 0 {
			windows.CloseHandle(h)
		}
		return false
	}
	if err != nil {
		return true // mutex creation itself failed: proceed unguarded, as HEAD does
	}
	singleInstanceMutex = h
	return true
}
