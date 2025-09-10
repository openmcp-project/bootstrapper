package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// defaultIgnoredDirs contains directories that should be ignored during comparison
var defaultIgnoredDirs = []string{
	".git",
	".gitignore",
}

// shouldIgnoreDir checks if a directory should be ignored
func shouldIgnoreDir(dirName string, ignoredDirs []string) bool {
	for _, ignored := range ignoredDirs {
		if dirName == ignored {
			return true
		}
	}
	return false
}

// AssertDirectoriesEqual compares two directories recursively and checks if all expected files
// are present and their content matches. It fails the test if there are any differences.
// It ignores common directories like .git, .DS_Store, etc.
func AssertDirectoriesEqual(t *testing.T, expectedDir, actualDir string) {
	AssertDirectoriesEqualWithIgnore(t, expectedDir, actualDir, defaultIgnoredDirs)
}

// AssertDirectoriesEqualWithIgnore compares two directories recursively with custom ignore list
func AssertDirectoriesEqualWithIgnore(t *testing.T, expectedDir, actualDir string, ignoredDirs []string) {
	t.Helper()

	// Collect all files from both directories
	expectedFiles := make(map[string][]byte)
	actualFiles := make(map[string][]byte)

	// Walk expected directory
	err := filepath.Walk(expectedDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check if we should ignore this directory
		if info.IsDir() && shouldIgnoreDir(info.Name(), ignoredDirs) {
			return filepath.SkipDir
		}

		if !info.IsDir() {
			relPath, err := filepath.Rel(expectedDir, path)
			if err != nil {
				return err
			}

			// Skip files in ignored directories
			pathParts := strings.Split(relPath, string(filepath.Separator))
			for _, part := range pathParts {
				if shouldIgnoreDir(part, ignoredDirs) {
					return nil
				}
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			expectedFiles[relPath] = content
		}
		return nil
	})
	assert.NoError(t, err, "Failed to walk expected directory: %s", expectedDir)

	// Walk actual directory
	err = filepath.Walk(actualDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check if we should ignore this directory
		if info.IsDir() && shouldIgnoreDir(info.Name(), ignoredDirs) {
			return filepath.SkipDir
		}

		if !info.IsDir() {
			relPath, err := filepath.Rel(actualDir, path)
			if err != nil {
				return err
			}

			// Skip files in ignored directories
			pathParts := strings.Split(relPath, string(filepath.Separator))
			for _, part := range pathParts {
				if shouldIgnoreDir(part, ignoredDirs) {
					return nil
				}
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			actualFiles[relPath] = content
		}
		return nil
	})
	assert.NoError(t, err, "Failed to walk actual directory: %s", actualDir)

	// Compare file counts
	assert.Equal(t, len(expectedFiles), len(actualFiles), "Number of files should match between directories")

	// Compare each file
	for relPath, expectedContent := range expectedFiles {
		actualContent, exists := actualFiles[relPath]
		assert.True(t, exists, "File %s should exist in actual directory", relPath)
		if exists {
			assert.Equal(t, string(expectedContent), string(actualContent), "Content of file %s should match", relPath)
		}
	}

	// Check for unexpected files in actual directory
	for relPath := range actualFiles {
		_, exists := expectedFiles[relPath]
		assert.True(t, exists, "Unexpected file %s found in actual directory", relPath)
	}
}

// AssertDirectoriesEqualWithNormalization compares two directories recursively with content normalization
// for handling dynamic values like temporary paths, timestamps, etc.
func AssertDirectoriesEqualWithNormalization(t *testing.T, expectedDir, actualDir string, normalizer func(string, string) string) {
	AssertDirectoriesEqualWithNormalizationWithIgnore(t, expectedDir, actualDir, defaultIgnoredDirs, normalizer)
}

// AssertDirectoriesEqualWithNormalizationWithIgnore compares two directories recursively with content normalization
// and a custom ignore list
func AssertDirectoriesEqualWithNormalizationWithIgnore(t *testing.T, expectedDir, actualDir string, ignoredDirs []string, normalizer func(string, string) string) {
	t.Helper()

	// Collect all files from both directories
	expectedFiles := make(map[string][]byte)
	actualFiles := make(map[string][]byte)

	// Walk expected directory
	err := filepath.Walk(expectedDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check if we should ignore this directory
		if info.IsDir() && shouldIgnoreDir(info.Name(), defaultIgnoredDirs) {
			return filepath.SkipDir
		}

		if !info.IsDir() {
			relPath, err := filepath.Rel(expectedDir, path)
			if err != nil {
				return err
			}

			// Skip files in ignored directories
			pathParts := strings.Split(relPath, string(filepath.Separator))
			for _, part := range pathParts {
				if shouldIgnoreDir(part, defaultIgnoredDirs) {
					return nil
				}
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			expectedFiles[relPath] = content
		}
		return nil
	})
	assert.NoError(t, err, "Failed to walk expected directory: %s", expectedDir)

	// Walk actual directory
	err = filepath.Walk(actualDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check if we should ignore this directory
		if info.IsDir() && shouldIgnoreDir(info.Name(), ignoredDirs) {
			return filepath.SkipDir
		}

		if !info.IsDir() {
			relPath, err := filepath.Rel(actualDir, path)
			if err != nil {
				return err
			}

			// Skip files in ignored directories
			pathParts := strings.Split(relPath, string(filepath.Separator))
			for _, part := range pathParts {
				if shouldIgnoreDir(part, ignoredDirs) {
					return nil
				}
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			actualFiles[relPath] = content
		}
		return nil
	})
	assert.NoError(t, err, "Failed to walk actual directory: %s", actualDir)

	// Compare file counts
	assert.Equal(t, len(expectedFiles), len(actualFiles), "Number of files should match between directories")

	// Compare each file with normalization
	for relPath, expectedContent := range expectedFiles {
		actualContent, exists := actualFiles[relPath]
		assert.True(t, exists, "File %s should exist in actual directory", relPath)
		if exists {
			normalizedExpected := normalizer(string(expectedContent), relPath)
			normalizedActual := normalizer(string(actualContent), relPath)
			assert.Equal(t, normalizedExpected, normalizedActual, "Content of file %s should match after normalization", relPath)
		}
	}

	// Check for unexpected files in actual directory
	for relPath := range actualFiles {
		_, exists := expectedFiles[relPath]
		assert.True(t, exists, "Unexpected file %s found in actual directory", relPath)
	}
}

// AssertDirectoryContains checks if the actual directory contains all files from the expected directory
// and their content matches. It allows extra files in the actual directory that are not in expected.
func AssertDirectoryContains(t *testing.T, expectedDir, actualDir string) {
	t.Helper()

	// Collect all files from expected directory
	expectedFiles := make(map[string][]byte)

	// Walk expected directory
	err := filepath.Walk(expectedDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			relPath, err := filepath.Rel(expectedDir, path)
			if err != nil {
				return err
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			expectedFiles[relPath] = content
		}
		return nil
	})
	assert.NoError(t, err, "Failed to walk expected directory: %s", expectedDir)

	// Check each expected file exists in actual directory
	for relPath, expectedContent := range expectedFiles {
		actualPath := filepath.Join(actualDir, relPath)
		actualContent, err := os.ReadFile(actualPath)
		assert.NoError(t, err, "File %s should exist in actual directory", relPath)
		if err == nil {
			assert.Equal(t, string(expectedContent), string(actualContent), "Content of file %s should match", relPath)
		}
	}
}

// AssertFileExists checks if a file exists at the specified path within a directory
func AssertFileExists(t *testing.T, dir, relativeFilePath string) {
	t.Helper()

	fullPath := filepath.Join(dir, relativeFilePath)
	_, err := os.Stat(fullPath)
	assert.NoError(t, err, "File %s should exist in directory %s", relativeFilePath, dir)
}

// AssertFileContent checks if a file exists and has the expected content
func AssertFileContent(t *testing.T, dir, relativeFilePath, expectedContent string) {
	t.Helper()

	fullPath := filepath.Join(dir, relativeFilePath)
	actualContent, err := os.ReadFile(fullPath)
	assert.NoError(t, err, "Should be able to read file %s", relativeFilePath)
	if err == nil {
		assert.Equal(t, expectedContent, string(actualContent), "Content of file %s should match", relativeFilePath)
	}
}

// WriteToFile writes content to a file at the specified path
func WriteToFile(t *testing.T, filePath, content string) {
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write to file %s: %v", filePath, err)
	}
}

// ReadFromFile reads content from a file at the specified path
func ReadFromFile(t *testing.T, filePath string) string {
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read from file %s: %v", filePath, err)
	}
	return string(data)
}
