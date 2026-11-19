//go:build !windows

package alarm

func hasOutputDevice() bool { return true }
