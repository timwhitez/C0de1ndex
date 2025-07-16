package llm

import (
	"C0de1ndex/internal/config"
	"C0de1ndex/internal/prompts"
	"C0de1ndex/internal/types"
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"

	openai "github.com/sashabaranov/go-openai"
)

// LLMClient wraps the OpenAI client and prompt templates.
// LLMClient 封装了 OpenAI 客户端和 prompt 模板。
type LLMClient struct {
	Client  OpenAIClient // Use the interface, not the concrete client
	Prompts *prompts.Prompts
	Debug   bool
}

// NewLLMClient creates a new client for interacting with the LLM.
// NewLLMClient 创建一个新的用于与 LLM 交互的客户端。
func NewLLMClient(config *config.Config, prompts *prompts.Prompts) (*LLMClient, error) {
	if config.OpenAIApiKey == "" {
		return nil, fmt.Errorf("OpenAI API key is missing. Please set it in your .env file or as an environment variable")
	}

	clientConfig := openai.DefaultConfig(config.OpenAIApiKey)
	if config.OpenAIBaseURL != "" {
		clientConfig.BaseURL = config.OpenAIBaseURL
	}

	client := openai.NewClientWithConfig(clientConfig)

	return &LLMClient{
		Client:  client, // The concrete client satisfies the interface
		Prompts: prompts,
	}, nil
}

// FilterFilesWithLLM sends the file list to the LLM and returns the filtered list.
// FilterFilesWithLLM 将文件列表发送给 LLM 并返回过滤后的列表。
func (c *LLMClient) FilterFilesWithLLM(allFiles []string, gitignorePatterns []string, model string) ([]string, error) {
	var filteredFiles []string
	var err error

	for i := 0; i < 3; i++ {
		prompt, err := c.renderFilterPrompt(allFiles, gitignorePatterns)
		if err != nil {
			return nil, fmt.Errorf("error rendering filter prompt: %w", err)
		}

		// Debug output
		if c.Debug {
			fmt.Println("\n--- LLM Prompt for Filtering ---")
			fmt.Println(prompt)
		}

		// Create the chat completion request
		req := openai.ChatCompletionRequest{
			Model: model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: c.Prompts.FilterSystemPrompt,
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
			return nil, fmt.Errorf("error from OpenAI API after retries: %w", err)
		}

		if len(resp.Choices) == 0 {
			return nil, fmt.Errorf("received no response choices from LLM")
		}

		// Extract the JSON content from the response
		jsonResponse := resp.Choices[0].Message.Content
		if c.Debug {
			fmt.Printf("\n--- Received LLM Response ---\n%s\n", jsonResponse)
		}

		// Parse the JSON response using the repair utility
		parsedResponse, err := RepairAndParseJSON[struct {
			FilteredFiles []string `json:"filtered_files"`
		}](jsonResponse)
		if err == nil {
			filteredFiles = parsedResponse.FilteredFiles
			if !c.Debug {
				fmt.Printf("\nLLM has filtered files, %d files remaining.\n", len(filteredFiles))
			}
			break
		}

		fmt.Printf("Retrying due to JSON parsing error (%d/3)..\n", i+1)
	}

	return filteredFiles, err
}

// renderFilterPrompt populates the filter prompt template with data.
// renderFilterPrompt 使用数据填充过滤 prompt 模板。
func (c *LLMClient) renderFilterPrompt(allFiles []string, gitignorePatterns []string) (string, error) {
	var buf bytes.Buffer
	tmpl, err := template.New("filter").Parse(c.Prompts.FilterPrompt)
	if err != nil {
		return "", err
	}

	data := struct {
		AllFiles          string
		GitignorePatterns string
	}{
		AllFiles:          strings.Join(allFiles, "\n"),
		GitignorePatterns: strings.Join(gitignorePatterns, "\n"),
	}

	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (c *LLMClient) GenerateProjectSummary(analyses []*types.FileAnalysis, model string) (*types.ProjectSummary, error) {
	var projectSummary *types.ProjectSummary
	var err error

	for i := 0; i < 3; i++ {
		// Create a single string containing all the file summaries.
		var summaries strings.Builder
		for _, analysis := range analyses {
			summaries.WriteString(fmt.Sprintf("File: %s\nSummary: %s\n\n", analysis.FilePath, analysis.Summary))
		}

		prompt, err := c.renderProjectSummaryPrompt(summaries.String())
		if err != nil {
			return nil, fmt.Errorf("error rendering project summary prompt: %w", err)
		}

		// Debug output
		if c.Debug {
			fmt.Println("\n--- LLM Prompt for Project Summary ---")
			fmt.Println(prompt)
		}

		req := openai.ChatCompletionRequest{
			Model: model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: c.Prompts.ProjectSummarySystemPrompt,
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
			return nil, fmt.Errorf("error from OpenAI API for project summary: %w", err)
		}

		if len(resp.Choices) == 0 {
			return nil, fmt.Errorf("received no response choices from LLM for project summary")
		}

		jsonResponse := resp.Choices[0].Message.Content
		if c.Debug {
			fmt.Printf("\n--- Received LLM Project Summary ---\n%s\n", jsonResponse)
		}

		projectSummary, err = RepairAndParseJSON[types.ProjectSummary](jsonResponse)
		if err == nil {
			if !c.Debug {
				fmt.Println("\nProject summary generated.")
			}
			break
		}

		fmt.Printf("Retrying due to JSON parsing error (%d/3)..\n", i+1)
	}

	return projectSummary, err
}

func (c *LLMClient) renderProjectSummaryPrompt(summaries string) (string, error) {
	var buf bytes.Buffer
	tmpl, err := template.New("summary").Parse(c.Prompts.ProjectSummaryPrompt)
	if err != nil {
		return "", err
	}

	data := struct{ FileSummaries string }{FileSummaries: summaries}

	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
