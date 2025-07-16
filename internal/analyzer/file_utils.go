package analyzer

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// ListFiles recursively lists all files in a directory, returning their paths relative to the root.
// It skips directories and files starting with '.', which are common for version control, IDEs, and temporary files.
// It also skips some predefined files like executables, .env, and prompts.json.
func ListFiles(root string) ([]string, error) {
	var files []string

	exePath, _ := os.Executable()
	exeName := filepath.Base(exePath)

	ignoreList := map[string]bool{
		"prompts.json":  true,
		".env":          true,
		exeName:         true,
		"C0de1ndex":     true,
		"C0de1ndex.exe": true,
	}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		name := info.Name()

		// Skip hidden files and directories, but not "."
		if name != "." && strings.HasPrefix(name, ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip temporary files and directories
		if strings.HasPrefix(name, "~") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if !info.IsDir() {
			if ignoreList[name] {
				return nil
			}

			relPath, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			files = append(files, relPath)
		}
		return nil
	})
	return files, err
}

// ParseGitignore reads and parses a .gitignore file from the specified root directory.
// It returns a list of patterns.
func ParseGitignore(root string) ([]string, error) {
	path := filepath.Join(root, ".gitignore")
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // .gitignore not found is not an error
		}
		return nil, err
	}
	defer file.Close()

	var patterns []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Ignore comments and empty lines
		if line != "" && !strings.HasPrefix(line, "#") {
			patterns = append(patterns, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return patterns, nil
}
