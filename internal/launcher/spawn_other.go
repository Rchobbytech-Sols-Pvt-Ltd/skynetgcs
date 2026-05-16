//go:build !windows

package launcher

import "os/exec"

func startProcess(exePath, dir string, env []string, _ bool) (*childProcess, error) {
	cmd := exec.Command(exePath)
	cmd.Dir = dir
	cmd.Env = env

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return &childProcess{
		pid:  cmd.Process.Pid,
		kill: cmd.Process.Kill,
		wait: cmd.Wait,
	}, nil
}
