package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

type AgentScoreResult struct {
	SessionID    string `json:"session_id"`
	FinishReason string `json:"finish_reason"`
	Text         string `json:"text"`
	RequestID    string `json:"request_id"`
}

func AgentMathScore(question, answer string) (*AgentScoreResult, error) {
	apiKey := os.Getenv("DASHSCOPE_API_KEY")
	appId := os.Getenv("DASHSCOPE_MATH_APP_ID")

	if apiKey == "" {
		log.Fatal("请确保设置了DASHSCOPE_API_KEY。")
		return nil, fmt.Errorf("DASHSCOPE_API_KEY is not set")
	}

	url := fmt.Sprintf("https://dashscope.aliyuncs.com/api/v1/apps/%s/completion", appId)

	requestBody := map[string]interface{}{
		"input": map[string]interface{}{
			"prompt": "好好批卷",
			"biz_params": map[string]interface{}{
				"imagepathquestion": question,
				"imagepathas":       answer,
			},
		},
		"parameters": map[string]interface{}{},
		"debug":      map[string]interface{}{},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal JSON: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("Failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Request failed with status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Output struct {
			FinishReason string `json:"finish_reason"`
			SessionID    string `json:"session_id"`
			Text         string `json:"text"`
		} `json:"output"`
		RequestID string `json:"request_id"`
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse response: %v", err)
	}

	// Map the parsed response to the result struct
	return &AgentScoreResult{
		SessionID:    result.Output.SessionID,
		FinishReason: result.Output.FinishReason,
		Text:         result.Output.Text,
		RequestID:    result.RequestID,
	}, nil
}
