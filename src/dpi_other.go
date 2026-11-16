//go:build !windows

package main

func systemScale() float64 { return 1 }

func resizeNativeWindow(handle uintptr, width, height int) {}
