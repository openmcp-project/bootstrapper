package utils

import (
	"bufio"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"
)

var (
	CacheDirRoot = filepath.Join(os.TempDir(), "openmcp-bootstrapper-test")
	OCMVersion   = ""
)

// DownloadOCMAndAddToPath downloads the OCM cli for the current platform and puts it to the PATH of the test
func DownloadOCMAndAddToPath(t *testing.T) {
	t.Helper()

	ocmVersion := getOCMVersion(t)

	cacheDir := filepath.Join(CacheDirRoot, "ocm-cli-cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatalf("failed to create cache dir: %v", err)
	}

	ocmBinaryName := "ocm-" + ocmVersion + "-" + runtime.GOOS + "-" + runtime.GOARCH
	ocmPath := filepath.Join(cacheDir, ocmBinaryName)

	if _, err := os.Stat(ocmPath); os.IsNotExist(err) {
		t.Log("Downloading OCM as it is not present in the cache directory, starting download...")

		downloadURL := "https://github.com/open-component-model/ocm/releases/download/v" +
			ocmVersion + "/ocm-" + ocmVersion + "-" + runtime.GOOS + "-" + runtime.GOARCH + ".tar.gz"

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

		// if symlink already exists, remove it
		symlinkPath := filepath.Join(cacheDir, "ocm")
		if _, err := os.Lstat(symlinkPath); err == nil {
			if err := os.Remove(symlinkPath); err != nil {
				t.Fatalf("failed to remove existing symlink: %v", err)
			}
		} else if !os.IsNotExist(err) {
			t.Fatalf("failed to check existing symlink: %v", err)
		}

		// create symlink to the ocm binary
		if err := os.Symlink(ocmPath, symlinkPath); err != nil {
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
		"--skip-digest-generation",
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

func getOCMVersion(t *testing.T) string {
	var err error

	if OCMVersion != "" {
		t.Logf("Using cached OCM_VERSION: %s", OCMVersion)
		return OCMVersion
	}

	// Find the parent directory containing the Dockerfile
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	var (
		dockerfilePath string
		currentDir     = cwd
	)

	for {
		dockerfilePath = filepath.Join(currentDir, "Dockerfile")
		if _, err = os.Stat(dockerfilePath); err == nil {
			break
		} else {
			if !os.IsNotExist(err) {
				t.Fatalf("failed to check Dockerfile existence: %v", err)
			}
		}
		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			t.Fatalf("Dockerfile not found in any parent directory of %s", cwd)
		}

		currentDir = parent
	}
	OCMVersion = parseDockerfileOCMVersion(dockerfilePath, t)

	t.Logf("Parsed OCM_VERSION from Dockerfile: %s", OCMVersion)
	return OCMVersion
}

// ParseDockerfileOCMVersion parses the Dockerfile to extract the OCM_VERSION argument value.
func parseDockerfileOCMVersion(dockerfilePath string, t *testing.T) string {
	file, err := os.Open(dockerfilePath)
	if err != nil {
		t.Fatalf("failed to open Dockerfile: %v", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			t.Fatalf("failed to close Dockerfile: %v", err)
		}
	}(file)

	scanner := bufio.NewScanner(file)
	re := regexp.MustCompile(`^ARG OCM_VERSION=([\w.-]+)`)
	for scanner.Scan() {
		line := scanner.Text()
		matches := re.FindStringSubmatch(line)
		if len(matches) == 2 {
			return matches[1]
		}
	}
	if err = scanner.Err(); err != nil {
		t.Fatalf("failed to read Dockerfile: %v", err)
	}

	t.Fatalf("OCM_VERSION not found in Dockerfile: %s", dockerfilePath)
	return ""
}
