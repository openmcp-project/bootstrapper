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

var (
	CacheDirRoot = filepath.Join(os.TempDir(), "openmcp-bootstrapper-test")
)

// DownloadOCMAndAddToPath downloads the OCM cli for the current platform and puts it to the PATH of the test
func DownloadOCMAndAddToPath(t *testing.T) {
	t.Helper()

	cacheDir := filepath.Join(CacheDirRoot, "ocm-cli-cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatalf("failed to create cache dir: %v", err)
	}

	ocmBinaryName := "ocm-" + OCM_VERSION + "-" + runtime.GOOS + "-" + runtime.GOARCH
	ocmPath := filepath.Join(cacheDir, ocmBinaryName)

	if _, err := os.Stat(ocmPath); os.IsNotExist(err) {
		t.Log("Downloading OCM as it is not present in the cache directory, starting download...")

		downloadURL := "https://github.com/open-component-model/ocm/releases/download/v" +
			OCM_VERSION + "/ocm-" + OCM_VERSION + "-" + runtime.GOOS + "-" + runtime.GOARCH + ".tar.gz"

		tempDir := t.TempDir()
		archivePath := filepath.Join(tempDir, "ocm.tar.gz")
		out, err := os.Create(archivePath)
		if err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
		defer func(out *os.File) {
			err := out.Close()
			if err != nil {
				t.Fatalf("failed to close file: %v", err)
			}
		}(out)

		resp, err := http.Get(downloadURL)
		if err != nil {
			t.Fatalf("failed to download ocm: %v", err)
		}
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				t.Fatalf("failed to close response body: %v", err)
			}
		}(resp.Body)

		_, err = io.Copy(out, resp.Body)
		if err != nil {
			t.Fatalf("failed to save ocm: %v", err)
		}

		// Extract the tar.gz
		cmd := exec.Command("tar", "-xzf", archivePath, "-C", tempDir)
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to extract ocm: %v", err)
		}

		// Move the ocm binary to the cache dir
		binPath := filepath.Join(tempDir, "ocm")
		if _, err := os.Stat(binPath); err != nil {
			t.Fatalf("ocm binary not found after extraction: %v", err)
		}
		if err := os.Rename(binPath, ocmPath); err != nil {
			t.Fatalf("failed to move ocm binary to cache: %v", err)
		}
		if err := os.Chmod(ocmPath, 0o755); err != nil {
			t.Fatalf("failed to chmod ocm binary: %v", err)
		}

		if err := os.Symlink(ocmPath, filepath.Join(cacheDir, "ocm")); err != nil {
			t.Fatalf("failed to create symlink for ocm binary: %v", err)
		}
	} else {
		t.Log("OCM binary already exists in the cache directory, skipping download.")
	}

	// Prepend the cache dir to PATH
	err := os.Setenv("PATH", cacheDir+":"+os.Getenv("PATH"))
	if err != nil {
		t.Fatalf("failed to set PATH environment variable: %v", err)
	}
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
