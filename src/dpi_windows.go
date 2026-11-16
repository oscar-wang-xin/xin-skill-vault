//go:build windows

package main

import "golang.org/x/sys/windows"

var (
	user32              = windows.NewLazySystemDLL("user32.dll")
	procGetDpiForSystem = user32.NewProc("GetDpiForSystem")
	procSetWindowPos    = user32.NewProc("SetWindowPos")
)

const (
	swpNoMove     = 0x0002
	swpNoZOrder   = 0x0004
	swpNoActivate = 0x0010
)

func systemScale() float64 {
	dpi, _, _ := procGetDpiForSystem.Call()
	if dpi == 0 {
		return 1
	}
	return float64(dpi) / 96.0
}

func resizeNativeWindow(handle uintptr, width, height int) {
	procSetWindowPos.Call(handle, 0, 0, 0, uintptr(width), uintptr(height),
		swpNoMove|swpNoZOrder|swpNoActivate)
}
