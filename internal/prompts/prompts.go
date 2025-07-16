package prompts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/template"
)

// Prompts holds the templates for LLM interactions.
// Prompts 保存了与 LLM 交互的模板。
type Prompts struct {
	FilterSystemPrompt         string `json:"filter_system_prompt"`
	AnalysisSystemPrompt       string `json:"analysis_system_prompt"`
	ProjectSummarySystemPrompt string `json:"project_summary_system_prompt"`
	ChunkAnalysisSystemPrompt  string `json:"chunk_analysis_system_prompt"`
	FilterPrompt               string `json:"filter_prompt"`
	AnalysisPrompt             string `json:"analysis_prompt"`
	ProjectSummaryPrompt       string `json:"project_summary_prompt"`
	ChunkAnalysisPrompt        string `json:"chunk_analysis_prompt"`
	ConsolidationPrompt        string `json:"consolidation_prompt"`
}

// LoadPrompts reads the prompt templates from a JSON file.
// LoadPrompts 从 JSON 文件中读取 prompt 模板。
func LoadPrompts(path string) (*Prompts, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var prompts Prompts
	if err := json.Unmarshal(data, &prompts); err != nil {
		return nil, err
	}

	return &prompts, nil
}

func (p *Prompts) GetProjectSummaryPrompt(fileSummaries string) (string, error) {
	// Replace Windows-style backslashes with forward slashes for consistency
	fileSummaries = strings.ReplaceAll(fileSummaries, "\\\\", "/")
	fileSummaries = strings.ReplaceAll(fileSummaries, "\\", "/")
	return p.render("project_summary_prompt", map[string]string{"FileSummaries": fileSummaries})
}

func (p *Prompts) render(promptName string, data map[string]string) (string, error) {
	var promptText string
	switch promptName {
	case "project_summary_prompt":
		promptText = p.ProjectSummaryPrompt
	// Add other cases for different prompts if necessary
	default:
		return "", fmt.Errorf("prompt '%s' not found", promptName)
	}

	tmpl, err := template.New(promptName).Parse(promptText)
	if err != nil {
		return "", fmt.Errorf("failed to parse prompt template '%s': %w", promptName, err)
	}

	var rendered bytes.Buffer
	if err := tmpl.Execute(&rendered, data); err != nil {
		return "", fmt.Errorf("failed to execute prompt template '%s': %w", promptName, err)
	}

	return rendered.String(), nil
}
