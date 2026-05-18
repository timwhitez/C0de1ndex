package analyzer

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type ScanOptions struct {
	GitignorePatterns []string
	ExcludePatterns   []string
	IncludeExtensions []string
}

type ignorePattern struct {
	pattern  string
	negated  bool
	dirOnly  bool
	hasSlash bool
	regex    *regexp.Regexp
}

var builtInExcludePatterns = []string{
	"node_modules/", "target/", "build/", "dist/", "vendor/", ".venv/", "__pycache__/",
	".git/", ".idea/", ".vscode/", ".DS_Store", "*.lock", "*.log", "*.zip",
	"*.tar.gz", "*.bin", "*.so", "*.dll", "*.db", "*.sqlite",
	"*.min.js", "*.min.css", "*.exe", "*.jar", "*.war", "*.class",
	"coverage/", "tmp/", "temp/", ".cache/", ".npm/", ".yarn/",
}

// ListFiles recursively lists all files in a directory, returning their paths relative to the root.
// It skips directories and files starting with '.', which are common for version control, IDEs, and temporary files.
// It also skips some predefined files like executables, .env, and prompts.json.
func ListFiles(root string) ([]string, error) {
	return ScanFiles(root, ScanOptions{})
}

func ScanFiles(root string, opts ScanOptions) ([]string, error) {
	includeExts := normalizeExtensions(opts.IncludeExtensions)
	patterns := append([]string{}, opts.GitignorePatterns...)
	patterns = append(patterns, builtInExcludePatterns...)
	patterns = append(patterns, opts.ExcludePatterns...)
	compiledPatterns := compileIgnorePatterns(patterns)
	hasNegations := hasNegationPattern(compiledPatterns)
	ignoreList := scanIgnoreList()

	var files []string
	err := filepath.WalkDir(root, func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		name := d.Name()
		if name == "." {
			return nil
		}
		if strings.HasPrefix(name, "~") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		relPath, err := filepath.Rel(root, filePath)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)

		if d.IsDir() {
			if isIgnoredCompiled(relPath, true, compiledPatterns) && !hasNegations {
				return filepath.SkipDir
			}
			return nil
		}

		if ignoreList[name] || isIgnoredCompiled(relPath, false, compiledPatterns) || !extensionAllowed(relPath, includeExts) {
			return nil
		}

		files = append(files, relPath)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
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

	// Combine gitignore patterns with built-in patterns and command-line exclude patterns
	allPatterns := append(gitignorePatterns, builtInExcludePatterns...)
	if len(excludePatterns) > 0 {
		allPatterns = append(allPatterns, excludePatterns...)
	}
	compiledPatterns := compileIgnorePatterns(allPatterns)

	for _, file := range files {
		shouldInclude := true

		shouldInclude = !isIgnoredCompiled(file, false, compiledPatterns)

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
	compiled := compileIgnorePatterns([]string{pattern})
	if len(compiled) == 0 {
		return false
	}
	return compiled[0].matches(filepath.ToSlash(filePath), false)
}

func isIgnored(filePath string, isDir bool, patterns []string) bool {
	return isIgnoredCompiled(filePath, isDir, compileIgnorePatterns(patterns))
}

func isIgnoredCompiled(filePath string, isDir bool, patterns []ignorePattern) bool {
	ignored := false
	for _, pattern := range patterns {
		if pattern.matches(filePath, isDir) {
			ignored = !pattern.negated
		}
	}
	return ignored
}

func compileIgnorePatterns(patterns []string) []ignorePattern {
	compiled := make([]ignorePattern, 0, len(patterns))
	for _, pattern := range patterns {
		p, ok := compileIgnorePattern(pattern)
		if ok {
			compiled = append(compiled, p)
		}
	}
	return compiled
}

func compileIgnorePattern(pattern string) (ignorePattern, bool) {
	pattern = strings.TrimSpace(filepath.ToSlash(pattern))
	if pattern == "" || strings.HasPrefix(pattern, "#") {
		return ignorePattern{}, false
	}

	negated := strings.HasPrefix(pattern, "!")
	if negated {
		pattern = strings.TrimPrefix(pattern, "!")
	}
	dirOnly := strings.HasSuffix(pattern, "/")
	pattern = strings.TrimSuffix(pattern, "/")
	pattern = strings.TrimPrefix(pattern, "/")
	if pattern == "" {
		return ignorePattern{}, false
	}

	re, err := regexp.Compile("^" + globToRegex(pattern) + "$")
	if err != nil {
		return ignorePattern{}, false
	}
	return ignorePattern{
		pattern:  pattern,
		negated:  negated,
		dirOnly:  dirOnly,
		hasSlash: strings.Contains(pattern, "/"),
		regex:    re,
	}, true
}

func (p ignorePattern) matches(filePath string, isDir bool) bool {
	filePath = filepath.ToSlash(filePath)
	if p.dirOnly && !isDir && !strings.Contains(filePath, "/") && !p.regex.MatchString(filePath) {
		return false
	}

	if !p.hasSlash {
		return p.matchesPathPart(filePath, isDir)
	}

	if p.regex.MatchString(filePath) {
		return true
	}
	if p.dirOnly {
		return strings.HasPrefix(filePath, p.pattern+"/")
	}
	return false
}

func (p ignorePattern) matchesPathPart(filePath string, isDir bool) bool {
	parts := strings.Split(filePath, "/")
	for i, part := range parts {
		if p.regex.MatchString(part) {
			return !p.dirOnly || isDir || i < len(parts)-1
		}
	}
	return false
}

func pathMatchesGlob(value, pattern string) bool {
	re, err := regexp.Compile("^" + globToRegex(pattern) + "$")
	if err != nil {
		return value == pattern
	}
	return re.MatchString(value)
}

func globToRegex(pattern string) string {
	var b strings.Builder
	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				b.WriteString(".*")
				i++
			} else {
				b.WriteString("[^/]*")
			}
		case '?':
			b.WriteString("[^/]")
		case '.', '+', '(', ')', '|', '[', ']', '{', '}', '^', '$', '\\':
			b.WriteByte('\\')
			b.WriteByte(pattern[i])
		default:
			b.WriteByte(pattern[i])
		}
	}
	return b.String()
}

func normalizeExtensions(exts []string) map[string]struct{} {
	if len(exts) == 0 {
		return nil
	}
	result := make(map[string]struct{}, len(exts))
	for _, ext := range exts {
		ext = strings.TrimSpace(strings.ToLower(ext))
		if ext == "" {
			continue
		}
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		result[ext] = struct{}{}
	}
	return result
}

func extensionAllowed(filePath string, includeExts map[string]struct{}) bool {
	if len(includeExts) == 0 {
		return true
	}
	_, ok := includeExts[strings.ToLower(filepath.Ext(filePath))]
	return ok
}

func scanIgnoreList() map[string]bool {
	exePath, _ := os.Executable()
	exeName := filepath.Base(exePath)
	return map[string]bool{
		"prompts.json":  true,
		".env":          true,
		exeName:         true,
		"C0de1ndex":     true,
		"C0de1ndex.exe": true,
	}
}

func hasNegationPattern(patterns []ignorePattern) bool {
	for _, pattern := range patterns {
		if pattern.negated {
			return true
		}
	}
	return false
}
