//go:build 386

package winapi

// 32-bit user32 exports only the non-Ptr names.
var (
	getWindowLongPtr = user32.NewProc("GetWindowLongW")
	setWindowLongPtr = user32.NewProc("SetWindowLongW")
)
