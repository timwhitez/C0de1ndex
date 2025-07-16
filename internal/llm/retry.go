package llm

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

const (
	maxRetriesPerStage = 5
	initialWaitMs    = 2000
	sleepAfterFailures = 60 * time.Second
)

// ExecuteWithRetry wraps a function that calls the OpenAI API and retries it on failure.
// It performs two stages of retries.
// ExecuteWithRetry 包装一个调用 OpenAI API 的函数，并在失败时重试。它会执行两个阶段的重试。
func (c *LLMClient) ExecuteWithRetry(apiCall func() (openai.ChatCompletionResponse, error)) (openai.ChatCompletionResponse, error) {
	var resp openai.ChatCompletionResponse
	var err error

	for stage := 0; stage < 2; stage++ {
		wait := time.Duration(initialWaitMs) * time.Millisecond
		for i := 0; i < maxRetriesPerStage; i++ {
			resp, err = apiCall()
			if err == nil {
				return resp, nil // Success
			}

			// Check if the error is retryable
			var apiErr *openai.APIError
			if errors.As(err, &apiErr) {
				if apiErr.HTTPStatusCode == http.StatusTooManyRequests || apiErr.HTTPStatusCode >= 500 {
					fmt.Printf("API error (%d), retrying in %v... (Stage %d, Attempt %d/%d)\n", apiErr.HTTPStatusCode, wait, stage+1, i+1, maxRetriesPerStage)
					time.Sleep(wait)
					wait *= 2 // Exponential backoff
					continue
				}
			}

			// If the error is not an APIError or not a retryable status code, break the inner loop
			break
		}

		// If we have not succeeded after the first stage, sleep and then try again.
		if stage == 0 && err != nil {
			fmt.Printf("First stage of retries failed. Sleeping for %v before starting second stage.\n", sleepAfterFailures)
			time.Sleep(sleepAfterFailures)
		}
	}

	return resp, err // Return the last error after all retries have failed
}
