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

// FilterFilesByRules applies gitignore patterns and built-in rules to filter files.
// This function performs rule-based filtering before LLM filtering.
func FilterFilesByRules(files []string, gitignorePatterns []string, excludePatterns []string) []string {
	var filteredFiles []string

	// Built-in patterns for common files/directories to exclude
	builtInPatterns := []string{
		"node_modules", "target", "build", "dist", "vendor", ".venv", "__pycache__",
		".git", ".idea", ".vscode", ".DS_Store", "*.lock", "*.log", "*.zip",
		"*.tar.gz", "*.bin", "*.so", "*.dll", "*.db", "*.sqlite",
		"*.min.js", "*.min.css", "*.exe", "*.jar", "*.war", "*.class",
		"coverage", "tmp", "temp", ".cache", ".npm", ".yarn",
	}

	// Combine gitignore patterns with built-in patterns and command-line exclude patterns
	allPatterns := append(gitignorePatterns, builtInPatterns...)
	if len(excludePatterns) > 0 {
		allPatterns = append(allPatterns, excludePatterns...)
	}

	for _, file := range files {
		shouldInclude := true

		// Check against all patterns
		for _, pattern := range allPatterns {
			if matchesPattern(file, pattern) {
				shouldInclude = false
				break
			}
		}

		// Additional rule: exclude test files and fixtures unless they're core logic
		if shouldInclude && (strings.Contains(file, "test") || strings.Contains(file, "fixture")) {
			// Allow test files that might be core logic (e.g., main test files)
			if !strings.HasSuffix(file, "_test.go") && !strings.HasSuffix(file, ".test.js") &&
				!strings.Contains(file, "main") && !strings.Contains(file, "index") {
				shouldInclude = false
			}
		}

		if shouldInclude {
			filteredFiles = append(filteredFiles, file)
		}
	}

	return filteredFiles
}

// matchesPattern checks if a file path matches a gitignore-style pattern
func matchesPattern(filePath, pattern string) bool {
	// Normalize path separators
	filePath = filepath.ToSlash(filePath)
	pattern = filepath.ToSlash(pattern)
	
	// Handle directory patterns (ending with /)
	if strings.HasSuffix(pattern, "/") {
		pattern = strings.TrimSuffix(pattern, "/")
		return strings.Contains(filePath, pattern+"/") || strings.HasPrefix(filePath, pattern+"/")
	}

	// Handle patterns starting with /
	if strings.HasPrefix(pattern, "/") {
		pattern = strings.TrimPrefix(pattern, "/")
		return strings.HasPrefix(filePath, pattern)
	}

	// Handle wildcard patterns
	if strings.Contains(pattern, "*") {
		matched, _ := filepath.Match(pattern, filepath.Base(filePath))
		if matched {
			return true
		}
		// Also check full path for patterns like "*.log"
		matched, _ = filepath.Match(pattern, filePath)
		return matched
	}

	// Handle directory patterns without trailing slash (like "venv")
	// Check if the pattern matches any directory component in the path
	pathParts := strings.Split(filePath, "/")
	for _, part := range pathParts {
		if part == pattern {
			return true
		}
	}
	
	// Also check if the file is inside a directory that matches the pattern
	if strings.Contains(filePath, pattern+"/") || strings.HasPrefix(filePath, pattern+"/") {
		return true
	}

	// Handle exact matches and substring matches
	return strings.Contains(filePath, pattern) || filepath.Base(filePath) == pattern
}
