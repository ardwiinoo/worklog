package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func getLogDir() (string, error) {
	baseDir, err := getAppBaseDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(baseDir, logDirName), nil
}

func getAppBaseDir() (string, error) {
	exePath, err := os.Executable()
	if err == nil {
		exePath, err = filepath.Abs(exePath)
		if err == nil {
			exeDir := filepath.Dir(exePath)

			if !strings.Contains(strings.ToLower(exeDir), "go-build") {
				return exeDir, nil
			}
		}
	}

	return os.Getwd()
}

func openPath(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	switch runtime.GOOS {
	case "windows":
		return exec.Command("cmd", "/c", "start", "", absPath).Start()
	case "darwin":
		return exec.Command("open", absPath).Start()
	default:
		return exec.Command("xdg-open", absPath).Start()
	}
}
