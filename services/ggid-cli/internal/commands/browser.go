package commands

import (
	"os/exec"
	"runtime"
)

// openBrowserOS opens the default browser on the user's OS.
func openBrowserOS(url string) {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	default: // linux, freebsd, etc.
		cmd = "xdg-open"
		args = []string{url}
	}

	// Best-effort: ignore errors silently.
	_ = exec.Command(cmd, args...).Start()
}
