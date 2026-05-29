//go:build windows

package main

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// psapi.GetProcessMemoryInfo is not exposed by x/sys/windows, so bind it here.
var (
	modpsapi                 = windows.NewLazySystemDLL("psapi.dll")
	procGetProcessMemoryInfo = modpsapi.NewProc("GetProcessMemoryInfo")
)

// processMemoryCounters mirrors the Win32 PROCESS_MEMORY_COUNTERS struct.
type processMemoryCounters struct {
	cb                         uint32
	pageFaultCount             uint32
	peakWorkingSetSize         uintptr
	workingSetSize             uintptr
	quotaPeakPagedPoolUsage    uintptr
	quotaPagedPoolUsage        uintptr
	quotaPeakNonPagedPoolUsage uintptr
	quotaNonPagedPoolUsage     uintptr
	pagefileUsage              uintptr
	peakPagefileUsage          uintptr
}

// procWorkingSetMB returns the VPM process's resident working-set size in MiB.
// This is real physical RAM for the Go/Wails host process (it does not include
// the separate WebView2 child processes). Returns 0 on error so the caller can
// fall back to the Go heap estimate.
func procWorkingSetMB() float64 {
	var c processMemoryCounters
	c.cb = uint32(unsafe.Sizeof(c))
	r, _, _ := procGetProcessMemoryInfo.Call(
		uintptr(windows.CurrentProcess()),
		uintptr(unsafe.Pointer(&c)),
		uintptr(c.cb),
	)
	if r == 0 {
		return 0
	}
	return float64(c.workingSetSize) / 1024 / 1024
}
