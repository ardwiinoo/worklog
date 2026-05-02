package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func isAutoStartEnabled() bool {
	if runtime.GOOS != "windows" {
		return false
	}

	runKey := `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`

	cmd := exec.Command("reg", "query", runKey, "/v", autoStartName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

	text := strings.ToLower(string(output))

	return strings.Contains(text, strings.ToLower(autoStartName))
}

func setAutoStart(enable bool) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("auto-start is only supported on Windows for now")
	}

	runKey := `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`

	if !enable {
		cmd := exec.Command("reg", "delete", runKey, "/v", autoStartName, "/f")
		output, err := cmd.CombinedOutput()
		if err != nil {
			if isRegistryValueMissing(string(output)) {
				return nil
			}

			return fmt.Errorf("failed to disable auto-start: %s", string(output))
		}

		return nil
	}

	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	exePath, err = filepath.Abs(exePath)
	if err != nil {
		return err
	}

	value := fmt.Sprintf(`"%s"`, exePath)

	cmd := exec.Command(
		"reg",
		"add",
		runKey,
		"/v",
		autoStartName,
		"/t",
		"REG_SZ",
		"/d",
		value,
		"/f",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to enable auto-start: %s", string(output))
	}

	return nil
}

func isRegistryValueMissing(output string) bool {
	text := strings.ToLower(output)

	return strings.Contains(text, "unable to find") ||
		strings.Contains(text, "cannot find") ||
		strings.Contains(text, "not found") ||
		strings.Contains(text, "was unable to find")
}
