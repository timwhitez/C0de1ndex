package analyzer

import (
	"context"
	"sort"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/c"
	"github.com/smacker/go-tree-sitter/cpp"
	"github.com/smacker/go-tree-sitter/csharp"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/php"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/ruby"
	"github.com/smacker/go-tree-sitter/rust"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
)

var languageMappings = map[string]*sitter.Language{
	"Go":         golang.GetLanguage(),
	"Java":       java.GetLanguage(),
	"JavaScript": javascript.GetLanguage(),
	"Python":     python.GetLanguage(),
	"TypeScript": typescript.GetLanguage(),
	"C":          c.GetLanguage(),
	"C++":        cpp.GetLanguage(),
	"C#":         csharp.GetLanguage(),
	"Ruby":       ruby.GetLanguage(),
	"Rust":       rust.GetLanguage(),
	"PHP":        php.GetLanguage(),
}

// TreeSitterChunker is responsible for splitting code into chunks based on language-specific syntax trees.
type TreeSitterChunker struct{}

// NewTreeSitterChunker creates a new TreeSitterChunker.
func NewTreeSitterChunker() *TreeSitterChunker {
	return &TreeSitterChunker{}
}

// Split takes code as a string and the language, and returns a slice of strings, where each string is a chunk.
func (chunker *TreeSitterChunker) Split(code, language string) ([]string, error) {
	parser := sitter.NewParser()
	var lang *sitter.Language
	var query *sitter.Query

	switch language {
	case "Go":
		lang = golang.GetLanguage()
		q, err := sitter.NewQuery([]byte(`(function_declaration) @func`), lang)
		if err != nil {
			return nil, err
		}
		query = q
	case "JavaScript":
		lang = javascript.GetLanguage()
		q, err := sitter.NewQuery([]byte(`(function_declaration) @func`), lang)
		if err != nil {
			return nil, err
		}
		query = q
	case "TypeScript":
		lang = typescript.GetLanguage()
		q, err := sitter.NewQuery([]byte(`(function_declaration) @func`), lang)
		if err != nil {
			return nil, err
		}
		query = q
	case "Python":
		lang = python.GetLanguage()
		q, err := sitter.NewQuery([]byte(`(function_definition) @func`), lang)
		if err != nil {
			return nil, err
		}
		query = q
	case "Java":
		lang = java.GetLanguage()
		q, err := sitter.NewQuery([]byte(`(method_declaration) @func`), lang)
		if err != nil {
			return nil, err
		}
		query = q
	case "C":
		lang = c.GetLanguage()
		q, err := sitter.NewQuery([]byte(`(function_definition) @func`), lang)
		if err != nil {
			return nil, err
		}
		query = q
	case "C++":
		lang = cpp.GetLanguage()
		q, err := sitter.NewQuery([]byte(`(function_definition) @func`), lang)
		if err != nil {
			return nil, err
		}
		query = q
	case "C#":
		lang = csharp.GetLanguage()
		q, err := sitter.NewQuery([]byte(`(method_declaration) @func`), lang)
		if err != nil {
			return nil, err
		}
		query = q
	case "Ruby":
		lang = ruby.GetLanguage()
		q, err := sitter.NewQuery([]byte(`(method) @func`), lang)
		if err != nil {
			return nil, err
		}
		query = q
	case "Rust":
		lang = rust.GetLanguage()
		q, err := sitter.NewQuery([]byte(`(function_item) @func`), lang)
		if err != nil {
			return nil, err
		}
		query = q
	case "PHP":
		lang = php.GetLanguage()
		q, err := sitter.NewQuery([]byte(`(function_definition) @func`), lang)
		if err != nil {
			return nil, err
		}
		query = q
	default:
		// Fallback for unsupported languages
		return fallbackSplit(code), nil
	}

	parser.SetLanguage(lang)
	tree, err := parser.ParseCtx(context.Background(), nil, []byte(code))
	if err != nil {
		return nil, err
	}

	qc := sitter.NewQueryCursor()
	qc.Exec(query, tree.RootNode())

	var functionNodes []*sitter.Node
	for {
		m, ok := qc.NextMatch()
		if !ok {
			break
		}
		for _, c := range m.Captures {
			functionNodes = append(functionNodes, c.Node)
		}
	}

	// Sort nodes by start position to handle them in order
	sort.Slice(functionNodes, func(i, j int) bool {
		return functionNodes[i].StartByte() < functionNodes[j].StartByte()
	})

	var chunks []string
	var otherCode strings.Builder
	var lastEnd uint32 = 0

	for _, node := range functionNodes {
		start := node.StartByte()
		if lastEnd < start {
			otherCode.WriteString(code[lastEnd:start])
		}
		chunks = append(chunks, code[start:node.EndByte()])
		lastEnd = node.EndByte()
	}

	if lastEnd < uint32(len(code)) {
		otherCode.WriteString(code[lastEnd:])
	}

	// Add the combined "other" code as a single chunk if it's not just whitespace
	if strings.TrimSpace(otherCode.String()) != "" {
		chunks = append(chunks, otherCode.String())
	}

	return chunks, nil
}
