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

type AgentResult struct {
	SessionID    string `json:"session_id"`
	FinishReason string `json:"finish_reason"`
	Text         string `json:"text"`
	RequestID    string `json:"request_id"`
}

func AgentMathScore(question, answer string) (*AgentResult, error) {
	apiKey := os.Getenv("DASHSCOPE_API_KEY")
	appId := os.Getenv("DASHSCOPE_MATH_APP_ID")

	if apiKey == "" {
		log.Fatal("请确保设置了DASHSCOPE_API_KEY。")
		return nil, fmt.Errorf("DASHSCOPE_API_KEY is not set")
	}

	if appId == "" {
		log.Fatal("请确保设置了DASHSCOPE_MATH_APP_ID。")
		return nil, fmt.Errorf("DASHSCOPE_MATH_APP_ID is not set")

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
	return &AgentResult{
		SessionID:    result.Output.SessionID,
		FinishReason: result.Output.FinishReason,
		Text:         result.Output.Text,
		RequestID:    result.RequestID,
	}, nil
}

func RetryAgentRequest(appIdEnv string, bizParams map[string]interface{}, retries int) (*AgentResult, error) {
	var err error
	var result *AgentResult

	for i := 0; i < retries; i++ {
		result, err = AgentRequest(appIdEnv, bizParams)
		if err == nil {
			return result, nil
		}
		log.Printf("Retry %d/%d failed: %v", i+1, retries, err)
	}

	return nil, fmt.Errorf("All retries failed: %v", err)
}

func AgentRequest(appIdEnv string, bizParams map[string]interface{}) (*AgentResult, error) {
	apiKey := os.Getenv("DASHSCOPE_API_KEY")

	if apiKey == "" {
		log.Fatal("请确保设置了DASHSCOPE_API_KEY。")
		return nil, fmt.Errorf("DASHSCOPE_API_KEY is not set")
	}

	if appIdEnv == "" {
		log.Fatal(fmt.Sprintf("请确保设置了appIdEnv%s。", appIdEnv))
		return nil, fmt.Errorf("%s is not set", appIdEnv)
	}

	url := fmt.Sprintf("https://dashscope.aliyuncs.com/api/v1/apps/%s/completion", appIdEnv)

	requestBody := map[string]interface{}{
		"input": map[string]interface{}{
			"prompt":     "好好批卷",
			"biz_params": bizParams,
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

	return &AgentResult{
		SessionID:    result.Output.SessionID,
		FinishReason: result.Output.FinishReason,
		Text:         result.Output.Text,
		RequestID:    result.RequestID,
	}, nil
}
