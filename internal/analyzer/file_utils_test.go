package analyzer

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestScanFilesAppliesGitignoreBuiltinsAndIncludes(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".gitignore", "artifacts/\n!artifacts/keep.go\n*.log\n/docs/generated/**\n")
	writeFile(t, root, "src/main.go", "package main\n")
	writeFile(t, root, "src/debug.log", "debug\n")
	writeFile(t, root, "artifacts/app.js", "console.log('skip')\n")
	writeFile(t, root, "artifacts/keep.go", "package artifacts\n")
	writeFile(t, root, "node_modules/pkg/index.go", "package pkg\n")
	writeFile(t, root, "docs/generated/schema.go", "package generated\n")
	writeFile(t, root, ".github/workflows/ci.yml", "name: ci\n")
	writeFile(t, root, "prompts.json", "{}\n")
	writeFile(t, root, ".env", "OPENAI_API_KEY=x\n")

	files, err := ScanFiles(root, ScanOptions{
		GitignorePatterns: []string{"artifacts/", "!artifacts/keep.go", "*.log", "/docs/generated/**"},
		IncludeExtensions: []string{"go", ".yml"},
	})
	if err != nil {
		t.Fatalf("ScanFiles returned error: %v", err)
	}

	want := []string{".github/workflows/ci.yml", "artifacts/keep.go", "src/main.go"}
	if !reflect.DeepEqual(files, want) {
		t.Fatalf("files = %#v, want %#v", files, want)
	}
}

func TestFilterFilesByRulesUsesOrderedNegationAndNoSubstringFalsePositive(t *testing.T) {
	files := []string{
		"contest/main.go",
		"src/main.go",
		"artifacts/app.go",
		"artifacts/keep.go",
		"src/app.log",
	}

	got := FilterFilesByRules(files, []string{"artifacts/", "!artifacts/keep.go", "*.log"}, nil)
	want := []string{"contest/main.go", "src/main.go", "artifacts/keep.go"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("filtered files = %#v, want %#v", got, want)
	}
}

func writeFile(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
