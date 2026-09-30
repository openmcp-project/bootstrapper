package utils

import (
	"bufio"
	"fmt"
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

		downloadURL := "https://github.com/open-component-model/open-component-model/releases/download/v" +
			ocmVersion + "/ocm-" + runtime.GOOS + "-" + runtime.GOARCH

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
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("failed to download ocm from %s: HTTP %d", downloadURL, resp.StatusCode)
		}

		// Write to a unique temporary file and rename it atomically: test packages run in parallel,
		// and a shared path would let one package rewrite a binary another package is executing.
		out, err := os.CreateTemp(cacheDir, ocmBinaryName+".download-*")
		if err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
		tmpPath := out.Name()
		defer func() { _ = os.Remove(tmpPath) }()
		if _, err = io.Copy(out, resp.Body); err != nil {
			_ = out.Close()
			t.Fatalf("failed to save ocm: %v", err)
		}
		if err := out.Close(); err != nil {
			t.Fatalf("failed to close file: %v", err)
		}
		if err := os.Rename(tmpPath, ocmPath); err != nil {
			t.Fatalf("failed to move ocm binary to cache: %v", err)
		}
		if err := os.Chmod(ocmPath, 0o755); err != nil {
			t.Fatalf("failed to chmod ocm binary: %v", err)
		}
	} else {
		t.Log("OCM binary already exists in the cache directory, skipping download.")
	}

	// Point the "ocm" symlink at the pinned binary. Create it under a unique name and rename it over the
	// existing link, so parallel test packages never observe a missing link and a stale link from another
	// OCM version is always replaced.
	symlinkPath := filepath.Join(cacheDir, "ocm")
	tmpLink := filepath.Join(cacheDir, fmt.Sprintf("ocm.link-%d", os.Getpid()))
	_ = os.Remove(tmpLink)
	if err := os.Symlink(ocmPath, tmpLink); err != nil {
		t.Fatalf("failed to create symlink for ocm binary: %v", err)
	}
	if err := os.Rename(tmpLink, symlinkPath); err != nil {
		t.Fatalf("failed to replace symlink for ocm binary: %v", err)
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

	cmd := exec.Command("ocm",
		"add", "componentversions",
		"--repository", "ctf::"+ctfDir,
		"--constructor", componentConstructorLocation,
		"--skip-reference-digest-processing",
	)

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
