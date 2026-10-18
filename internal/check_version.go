package internal

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

const releasesURL = "https://api.github.com/repos/darknisfire/KakaoTalkAdBlock/releases/latest"

// curlExe is a variable so tests can point it at a missing binary.
var curlExe = curlPath()

// SystemDir returns the System32 directory; PATH is never searched.
func SystemDir() string {
	root := os.Getenv("SystemRoot")
	if root == "" {
		root = `C:\Windows`
	}
	return filepath.Join(root, "System32")
}

// curlPath returns the curl.exe bundled with Windows.
func curlPath() string {
	return filepath.Join(SystemDir(), "curl.exe")
}

func CheckLatestVersion(ctx context.Context, currentVersion string) (string, bool) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, curlExe,
		"-q", "-s", "-f", "-L", "--max-redirs", "3", "-m", "10",
		"-A", "KakaoTalkAdBlock/"+currentVersion,
		"-H", "Accept: application/vnd.github+json",
		releasesURL)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
	body, err := cmd.Output()
	if err != nil {
		return currentVersion, false
	}

	tagName, ok := parseLatestTag(body)
	if !ok {
		return currentVersion, false
	}

	if hasNewRelease(currentVersion, tagName) {
		return tagName, true
	}

	return currentVersion, false
}

// parseLatestTag returns the release tag, or false for error bodies such as a rate-limit response.
func parseLatestTag(body []byte) (string, bool) {
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return "", false
	}
	tag, ok := data["tag_name"].(string)
	if !ok || tag == "" {
		return "", false
	}
	return tag, true
}

func hasNewRelease(current, latest string) bool {
	v1Parts := strings.Split(current, ".")
	v2Parts := strings.Split(latest, ".")

	for i := 0; i < len(v1Parts) && i < len(v2Parts); i++ {
		v1Part, err := strconv.Atoi(v1Parts[i])
		if err != nil {
			return false
		}
		v2Part, err := strconv.Atoi(v2Parts[i])
		if err != nil {
			return false
		}

		if v1Part > v2Part {
			return false
		} else if v1Part < v2Part {
			return true
		}
	}
	return false
}
