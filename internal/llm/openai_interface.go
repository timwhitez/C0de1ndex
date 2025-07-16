package llm

import (
	"context"
	openai "github.com/sashabaranov/go-openai"
)

// OpenAIClient defines the interface for the OpenAI client.
// This allows for mocking the client in tests.
// OpenAIClient 定义了 OpenAI 客户端的接口，以便在测试中进行模拟。
type OpenAIClient interface {
	CreateChatCompletion(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error)
}