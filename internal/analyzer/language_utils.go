package analyzer

import (
	"log"
	"path/filepath"
	"strings"
)

// Language defines the properties of a programming language.
type Language struct {
	Name        string
	LineComment string
}

// languages maps file extensions to language properties.
var languages = map[string]Language{
	".go":    {Name: "Go"},
	".js":    {Name: "JavaScript"},
	".ts":    {Name: "TypeScript"},
	".py":    {Name: "Python"},
	".java":  {Name: "Java"},
	".c":     {Name: "C"},
	".h":     {Name: "C"},
	".cpp":   {Name: "C++"},
	".hpp":   {Name: "C++"},
	".cs":    {Name: "C#"},
	".rb":    {Name: "Ruby"},
	".rs":    {Name: "Rust"},
	".php":   {Name: "PHP"},
	".html":  {Name: "HTML"},
	".css":   {Name: "CSS"},
	".md":    {Name: "Markdown"},
	".sh":    {Name: "Shell"},
	".swift": {Name: "Swift"},
	".kt":    {Name: "Kotlin"},
	".scala": {Name: "Scala"},
}

// GetLanguageFromFile determines the programming language from a file path.
func GetLanguageFromFile(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	if lang, ok := languages[ext]; ok {
		return lang.Name
	}
	return "Text" // Default to Text if unknown
}

// SplitCodeIntoChunks splits a large file into smaller chunks based on function or class boundaries.
// It uses a tree-sitter based chunker for supported languages, and falls back to a simple line-based split.
func SplitCodeIntoChunks(content, language string) []string {
	chunker := NewTreeSitterChunker()
	chunks, err := chunker.Split(content, language)
	if err != nil {
		log.Printf("error splitting code with tree-sitter for language %s: %v. Falling back to simple split.", language, err)
		return fallbackSplit(content)
	}
	return chunks
}

func fallbackSplit(code string) []string {
	// Simple line-based splitting for unsupported languages.
	lines := strings.Split(code, "\n")
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
