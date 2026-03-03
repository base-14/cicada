package clipboard

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// Copy writes text to the system clipboard.
func Copy(text string) error {
	var name string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		name = "pbcopy"
	case "linux":
		name = "xclip"
		args = []string{"-selection", "clipboard"}
	case "windows":
		name = "clip"
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	cmd := exec.Command(name, args...)
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
