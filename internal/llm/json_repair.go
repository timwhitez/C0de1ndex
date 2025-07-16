package llm

import (
	"encoding/json"
	"fmt"
	"strings"
)

// RepairAndParseJSON attempts to parse a JSON string, and if it fails, it tries to repair it.
// It handles common LLM-induced errors like markdown code fences and truncation.
func RepairAndParseJSON[T any](jsonStr string) (*T, error) {
	var result T

	// First, try to unmarshal the string as is.
	if err := json.Unmarshal([]byte(jsonStr), &result); err == nil {
		return &result, nil // Success on the first try
	}

	// If it fails, try to repair it.
	fmt.Println("Initial JSON parsing failed, attempting to repair...")
	repairedStr := strings.TrimSpace(jsonStr)

	// Remove markdown fences if they exist
	if strings.HasPrefix(repairedStr, "```json") {
		repairedStr = strings.TrimPrefix(repairedStr, "```json")
	}
	if strings.HasPrefix(repairedStr, "```") {
		repairedStr = strings.TrimPrefix(repairedStr, "```")
	}
	if strings.HasSuffix(repairedStr, "```") {
		repairedStr = strings.TrimSuffix(repairedStr, "```")
	}

	repairedStr = strings.TrimSpace(repairedStr)

	// Attempt to unmarshal the repaired string.
	if err := json.Unmarshal([]byte(repairedStr), &result); err != nil {
		// If it still fails, try to find the last complete JSON object.
		lastBracket := strings.LastIndex(repairedStr, "}")
		if lastBracket != -1 {
			potentialJSON := repairedStr[:lastBracket+1]
			if err := json.Unmarshal([]byte(potentialJSON), &result); err == nil {
				fmt.Println("Successfully parsed JSON after truncation repair.")
				return &result, nil
			}
		}

		// If it still fails, return the original error plus context.
		return nil, fmt.Errorf("failed to parse JSON even after all repair attempts: %w", err)
	}

	fmt.Println("Successfully parsed JSON after basic repair.")
	return &result, nil
}
