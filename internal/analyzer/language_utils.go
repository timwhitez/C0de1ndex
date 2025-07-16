package analyzer

import (
	"path/filepath"
	"strings"
)

// languageMappings maps file extensions to programming language names.
// languageMappings 将文件扩展名映射到编程语言名称。
var languageMappings = map[string]string{
	".go":   "Go",
	".js":   "JavaScript",
	".ts":   "TypeScript",
	".py":   "Python",
	".java": "Java",
	".cs":   "C#",
	".cpp":  "C++",
	".c":    "C",
	".h":    "C Header",
	".rs":   "Rust",
	".rb":   "Ruby",
	".php":  "PHP",
	".html": "HTML",
	".css":  "CSS",
	".md":   "Markdown",
	".sh":   "Shell",
	".swift": "Swift",
	".kt": "Kotlin",
	".scala": "Scala",
}

// GetLanguageFromFile determines the programming language from a file path.
// GetLanguageFromFile 从文件路径确定编程语言。
func GetLanguageFromFile(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	if lang, ok := languageMappings[ext]; ok {
		return lang
	}
	return "Text" // Default to Text if unknown
}

// SplitCodeIntoChunks splits a large file into smaller chunks based on function or class boundaries.
// This is a simplified implementation and may not be perfect for all languages.
// SplitCodeIntoChunks 将大文件根据函数或类边界分割成小块。
// 这是一个简化的实现，可能不适用于所有语言。
func SplitCodeIntoChunks(content, language string) []string {
	// For now, we will use a simple line-based splitting for all languages.
	// A more sophisticated approach would use language-specific parsers.
	lines := strings.Split(content, "\n")
	var chunks []string
	var currentChunk strings.Builder

	for i, line := range lines {
		currentChunk.WriteString(line + "\n")
		if (i+1)%100 == 0 { // Create a chunk every 100 lines
			chunks = append(chunks, currentChunk.String())
			currentChunk.Reset()
		}
	}

	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}

	return chunks
}
