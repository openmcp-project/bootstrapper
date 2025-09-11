package flux_deployer

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"

	"github.com/openmcp-project/bootstrapper/internal/template"
)

// TemplateDirectory processes the template files in the specified directory and writes
// the rendered content to the corresponding files in the result directory.
func TemplateDirectory(templateDirectory, resultDirectory string, templateInput TemplateInput, log *logrus.Logger) error {
	log.Debug("Templating")

	templateDir, err := os.Open(templateDirectory)
	if err != nil {
		return fmt.Errorf("failed to open template directory: %w", err)
	}
	defer func() {
		if err = templateDir.Close(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "failed to close template directory: %v\n", err)
		}
	}()

	te := template.NewTemplateExecution().WithMissingKeyOption("zero")

	// Recursively walk through all files in the template directory
	err = filepath.WalkDir(templateDirectory, func(path string, d os.DirEntry, walkError error) error {
		var (
			errInWalk error

			templateFromFile []byte
			templateResult   []byte

			relativePath   string
			fileInWorkTree *os.File
		)

		if walkError != nil {
			return walkError
		}

		relativePath, errInWalk = filepath.Rel(templateDirectory, path)
		if errInWalk != nil {
			return fmt.Errorf("failed to get relative path for %s: %w", path, errInWalk)
		}
		pathInWorkTree := filepath.Join(resultDirectory, relativePath)

		if d.IsDir() {
			err = os.MkdirAll(pathInWorkTree, 0755)
			if err != nil {
				return fmt.Errorf("failed to create directory %s: %w", path, err)
			}
		} else {
			log.Debugf("Found template file: %s", relativePath)

			templateFromFile, errInWalk = os.ReadFile(path)
			if errInWalk != nil {
				return fmt.Errorf("failed to read template file %s: %w", relativePath, err)
			}

			templateResult, errInWalk = te.Execute(path, string(templateFromFile), templateInput.ValuesWrapper())
			if errInWalk != nil {
				return fmt.Errorf("failed to execute template %s: %w", relativePath, errInWalk)
			}

			fileInWorkTree, errInWalk = os.Create(pathInWorkTree)
			if errInWalk != nil {
				return fmt.Errorf("failed to open file in worktree %s: %w", relativePath, errInWalk)
			}
			defer func(pathInRepo *os.File) {
				err := pathInRepo.Close()
				if err != nil {
					_, _ = fmt.Fprintf(os.Stderr, "failed to close file in worktree %s: %v\n", relativePath, err)
				}
			}(fileInWorkTree)

			_, errInWalk = fileInWorkTree.Write(templateResult)
			if errInWalk != nil {
				return fmt.Errorf("failed to write to file in worktree %s: %w", relativePath, errInWalk)
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk template directory: %w", err)
	}

	log.Debug("Templating done")
	return nil
}
