package utils

import (
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

const (
	// renovate: datasource=github-releases depName=ocm packageName=open-component-model/ocm
	OCM_VERSION = "0.27.0"
)

// DownloadOCMAndAddToPath downloads the OCM cli for the current platform and puts it to the PATH of the test
func DownloadOCMAndAddToPath(t *testing.T) {
	t.Helper()

	downloadURL := "https://github.com/open-component-model/ocm/releases/download/v" +
		OCM_VERSION + "/ocm-" + OCM_VERSION + "-" + runtime.GOOS + "-" + runtime.GOARCH + ".tar.gz"

	tempDir := t.TempDir()
	archivePath := filepath.Join(tempDir, "ocm.tar.gz")
	out, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	defer out.Close()

	resp, err := http.Get(downloadURL)
	if err != nil {
		t.Fatalf("failed to download ocm: %v", err)
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		t.Fatalf("failed to save ocm: %v", err)
	}

	// Extract the tar.gz
	cmd := exec.Command("tar", "-xzf", archivePath, "-C", tempDir)
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to extract ocm: %v", err)
	}

	// Find the ocm binary
	ocmPath := filepath.Join(tempDir, "ocm")
	if _, err := os.Stat(ocmPath); err != nil {
		t.Fatalf("ocm binary not found after extraction: %v", err)
	}

	t.Setenv("PATH", tempDir+":"+os.Getenv("PATH"))
}

// BuildComponent builds the component for the specified componentConstructorLocation and returns the ctf out directory.
func BuildComponent(componentConstructorLocation string, t *testing.T) string {
	tempDir := t.TempDir()
	ctfDir := filepath.Join(tempDir, "ctf")

	cmd := exec.Command("ocm", []string{
		"add",
		"componentversions",
		"--create",
		"--file",
		ctfDir,
		componentConstructorLocation}...)

	out, err := cmd.CombinedOutput()
	t.Log("OCM Output:", string(out))
	if err != nil {
		t.Fatalf("failed to build component: %v", err)
	}

	return ctfDir
}
