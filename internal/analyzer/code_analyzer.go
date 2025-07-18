package analyzer

import (
	"C0de1ndex/internal/llm"
	"C0de1ndex/internal/types"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	openai "github.com/sashabaranov/go-openai"
)

// AnalyzeFileWithLLM sends a single file's content to the LLM for analysis.
func AnalyzeFileWithLLM(c *llm.LLMClient, filePath, fileContent, model string) (*types.FileAnalysis, error) {
	// If the file is large, split it into chunks and analyze each chunk.
	if len(fileContent) > 32000 { // If file content exceeds 32k, chunk it
		fmt.Printf("File %s is very large, splitting into chunks...\n", filePath)
		chunks := SplitCodeIntoChunks(fileContent, GetLanguageFromFile(filePath))
		var chunkAnalyses []*types.FileAnalysis

		for i, chunk := range chunks {
			fmt.Printf("Analyzing chunk %d of %d for file %s...\n", i+1, len(chunks), filePath)
			chunkAnalysis, err := AnalyzeChunkWithLLM(c, filePath, chunk, model, c.Prompts.ChunkAnalysisSystemPrompt)
			if err != nil {
				return nil, fmt.Errorf("error analyzing chunk %d for file %s: %w", i+1, filePath, err)
			}
			chunkAnalyses = append(chunkAnalyses, chunkAnalysis)
		}

		// Consolidate the analyses from all chunks
		return ConsolidateChunkAnalyses(c, chunkAnalyses, model)
	}

	// If the file is not large, analyze it as a whole.
	return AnalyzeChunkWithLLM(c, filePath, fileContent, model, c.Prompts.AnalysisSystemPrompt)
}

func AnalyzeChunkWithLLM(c *llm.LLMClient, filePath, chunk, model, systemPrompt string) (*types.FileAnalysis, error) {
	var analysis *types.FileAnalysis
	var err error

	for i := 0; i < 3; i++ {
		prompt, err := RenderAnalysisPrompt(c, filePath, chunk)
		if err != nil {
			return nil, fmt.Errorf("error rendering analysis prompt: %w", err)
		}

		// Debug output
		if c.Debug {
			fmt.Println("\n--- LLM Prompt for Analysis ---")
			fmt.Println(prompt)
		}

		req := openai.ChatCompletionRequest{
			Model: model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: systemPrompt,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			ResponseFormat: &openai.ChatCompletionResponseFormat{
				Type: openai.ChatCompletionResponseFormatTypeJSONObject,
			},
		}

		resp, err := c.ExecuteWithRetry(func() (openai.ChatCompletionResponse, error) {
			return c.Client.CreateChatCompletion(context.Background(), req)
		})

		if err != nil {
			return nil, fmt.Errorf("error from OpenAI API for file %s after retries: %w", filePath, err)
		}

		if len(resp.Choices) == 0 {
			return nil, fmt.Errorf("received no response choices from LLM for file %s", filePath)
		}

		jsonResponse := resp.Choices[0].Message.Content
		if c.Debug {
			fmt.Printf("\n--- Received LLM Analysis for: %s ---\n%s\n", filePath, jsonResponse)
		}

		// Parse the JSON response using the repair utility
		analysis, err = llm.RepairAndParseJSON[types.FileAnalysis](jsonResponse)
		if err == nil {
			break
		}

		fmt.Printf("Retrying due to JSON parsing error (%d/3)...\n", i+1)
	}

	return analysis, err
}

// ConsolidateChunkAnalyses sends multiple chunk analyses to the LLM to be consolidated into a single analysis.
func ConsolidateChunkAnalyses(c *llm.LLMClient, analyses []*types.FileAnalysis, model string) (*types.FileAnalysis, error) {
	var consolidatedAnalysis *types.FileAnalysis
	var err error

	// Marshal the chunk analyses into a single JSON string
	analysesJSON, err := json.Marshal(analyses)
	if err != nil {
		return nil, fmt.Errorf("error marshalling chunk analyses: %w", err)
	}

	// Render the consolidation prompt
	prompt, err := RenderConsolidationPrompt(c, string(analysesJSON))
	if err != nil {
		return nil, fmt.Errorf("error rendering consolidation prompt: %w", err)
	}

	// Make the request to the LLM
	req := openai.ChatCompletionRequest{
		Model: model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: c.Prompts.AnalysisSystemPrompt, // Use analysis system prompt for consolidation
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
	}

	resp, err := c.ExecuteWithRetry(func() (openai.ChatCompletionResponse, error) {
		return c.Client.CreateChatCompletion(context.Background(), req)
	})

	if err != nil {
		return nil, fmt.Errorf("error from OpenAI API for consolidation after retries: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("received no response choices from LLM for consolidation")
	}

	jsonResponse := resp.Choices[0].Message.Content
	if c.Debug {
		fmt.Printf("\n--- Received LLM Consolidation ---\n%s\n", jsonResponse)
	}

	// Parse the JSON response using the repair utility
	consolidatedAnalysis, err = llm.RepairAndParseJSON[types.FileAnalysis](jsonResponse)
	if err != nil {
		return nil, fmt.Errorf("error parsing consolidated analysis: %w", err)
	}

	return consolidatedAnalysis, nil
}

func RenderAnalysisPrompt(c *llm.LLMClient, filePath, fileContent string) (string, error) {
	var buf strings.Builder
	tmpl, err := template.New("analysis").Parse(c.Prompts.AnalysisPrompt)
	if err != nil {
		return "", err
	}

	data := struct {
		FilePath    string
		FileContent string
		Language    string
	}{
		FilePath:    filePath,
		FileContent: fileContent,
		Language:    GetLanguageFromFile(filePath),
	}

	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func RenderConsolidationPrompt(c *llm.LLMClient, chunkAnalysesJSON string) (string, error) {
	var buf strings.Builder
	tmpl, err := template.New("consolidation").Parse(c.Prompts.ConsolidationPrompt)
	if err != nil {
		return "", err
	}

	data := struct {
		ChunkAnalyses string
	}{
		ChunkAnalyses: chunkAnalysesJSON,
	}

	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
