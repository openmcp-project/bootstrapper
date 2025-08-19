package util

import (
	"fmt"
	"os"
)

const (
	TempDirPrefix = "openmcp.cloud.bootstrapper-"
)

func CreateTempDir() (string, error) {
	tempDir, err := os.MkdirTemp("", TempDirPrefix)
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}
	return tempDir, nil
}

func DeleteTempDir(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("failed to delete temporary directory %s: %w", path, err)
	}
	return nil
}
