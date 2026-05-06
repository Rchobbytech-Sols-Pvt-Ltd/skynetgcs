//go:build !windows

package launcher

import "os/exec"

func hideConsole(cmd *exec.Cmd) {}
