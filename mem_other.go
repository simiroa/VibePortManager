//go:build !windows

package main

// procWorkingSetMB has no portable implementation off Windows; callers fall
// back to the Go heap estimate when this returns 0.
func procWorkingSetMB() float64 { return 0 }
