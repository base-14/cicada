package clipboard

import (
	"os/exec"
	"runtime"
	"testing"
)

func TestCopy_NoError(t *testing.T) {
	// Skip if no clipboard command is available
	var cmd string
	switch runtime.GOOS {
	case "darwin":
		cmd = "pbcopy"
	case "linux":
		cmd = "xclip"
	default:
		cmd = "clip"
	}
	if _, err := exec.LookPath(cmd); err != nil {
		t.Skipf("clipboard command %q not available, skipping", cmd)
	}

	err := Copy("test clipboard content")
	if err != nil {
		t.Errorf("Copy() returned error: %v", err)
	}
}

func TestCopy_EmptyString(t *testing.T) {
	var cmd string
	switch runtime.GOOS {
	case "darwin":
		cmd = "pbcopy"
	case "linux":
		cmd = "xclip"
	default:
		cmd = "clip"
	}
	if _, err := exec.LookPath(cmd); err != nil {
		t.Skipf("clipboard command %q not available, skipping", cmd)
	}

	err := Copy("")
	if err != nil {
		t.Errorf("Copy(\"\") returned error: %v", err)
	}
}
