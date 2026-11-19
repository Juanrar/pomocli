//go:build windows

package alarm

import "golang.org/x/sys/windows"

// hasOutputDevice dice si Windows ve al menos un dispositivo de salida.
// oto no avisa cuando no hay ninguno: usa un driver nulo y calla.
func hasOutputDevice() bool {
	proc := windows.NewLazySystemDLL("winmm.dll").NewProc("waveOutGetNumDevs")
	if err := proc.Find(); err != nil {
		return true
	}
	n, _, _ := proc.Call()
	return n > 0
}
