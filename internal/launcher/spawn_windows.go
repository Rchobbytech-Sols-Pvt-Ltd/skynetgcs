//go:build windows

package launcher

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	createNewConsole         = 0x00000010
	createUnicodeEnvironment = 0x00000400
	startfUseShowWindow      = 0x00000001
	swHide                   = 0
	swShow                   = 5
)

func startProcess(exePath, dir string, env []string, showConsole bool) (*childProcess, error) {
	cmdLine, err := windows.UTF16PtrFromString(`"` + exePath + `"`)
	if err != nil {
		return nil, err
	}
	appName, err := windows.UTF16PtrFromString(exePath)
	if err != nil {
		return nil, err
	}
	cwd, err := windows.UTF16PtrFromString(dir)
	if err != nil {
		return nil, err
	}
	envBlock, err := createEnvBlock(env)
	if err != nil {
		return nil, err
	}

	show := uint16(swHide)
	if showConsole {
		show = swShow
	}
	startupInfo := windows.StartupInfo{
		Cb:         uint32(unsafe.Sizeof(windows.StartupInfo{})),
		Flags:      startfUseShowWindow,
		ShowWindow: show,
	}
	var processInfo windows.ProcessInformation

	err = windows.CreateProcess(
		appName,
		cmdLine,
		nil,
		nil,
		false,
		createNewConsole|createUnicodeEnvironment,
		&envBlock[0],
		cwd,
		&startupInfo,
		&processInfo,
	)
	if err != nil {
		return nil, err
	}
	windows.CloseHandle(processInfo.Thread)

	processHandle := processInfo.Process
	child := &childProcess{
		pid: int(processInfo.ProcessId),
	}
	child.kill = func() error {
		return windows.TerminateProcess(processHandle, 1)
	}
	child.wait = func() error {
		_, err := windows.WaitForSingleObject(processHandle, windows.INFINITE)
		if err != nil {
			_ = windows.CloseHandle(processHandle)
			return err
		}

		var exitCode uint32
		if err := windows.GetExitCodeProcess(processHandle, &exitCode); err != nil {
			_ = windows.CloseHandle(processHandle)
			return err
		}
		_ = windows.CloseHandle(processHandle)

		if exitCode != 0 {
			return fmt.Errorf("exit status %d", exitCode)
		}
		return nil
	}

	return child, nil
}

func createEnvBlock(env []string) ([]uint16, error) {
	if len(env) == 0 {
		return []uint16{0, 0}, nil
	}

	block := make([]uint16, 0)
	for _, item := range env {
		encoded, err := windows.UTF16FromString(item)
		if err != nil {
			return nil, err
		}
		block = append(block, encoded...)
	}
	block = append(block, 0)
	return block, nil
}
