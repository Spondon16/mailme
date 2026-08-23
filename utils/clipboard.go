package utils

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

func CopyToClipboard(text string) error {
	switch runtime.GOOS {
	case "darwin":
		return runCmd("pbcopy", text)
	case "linux":
		// Try wl-copy (Wayland), then xclip, then xsel, then clip.exe (WSL)
		if _, err := exec.LookPath("wl-copy"); err == nil {
			if err := runCmd("wl-copy", text); err == nil {
				return nil
			}
		}
		if _, err := exec.LookPath("xclip"); err == nil {
			if err := runCmd("xclip", text, "-selection", "clipboard"); err == nil {
				return nil
			}
		}
		if _, err := exec.LookPath("xsel"); err == nil {
			if err := runCmd("xsel", text, "--clipboard", "--input"); err == nil {
				return nil
			}
		}
		if _, err := exec.LookPath("clip.exe"); err == nil {
			if err := runCmd("clip.exe", text); err == nil {
				return nil
			}
		}
		return fmt.Errorf("no clipboard provider found or working (install wl-copy, xclip, or xsel)")
	case "windows":
		return runCmd("clip.exe", text)
	default:
		return fmt.Errorf("unsupported operating system for clipboard: %s", runtime.GOOS)
	}
}

func runCmd(name string, input string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = strings.NewReader(input)
	return cmd.Run()
}
