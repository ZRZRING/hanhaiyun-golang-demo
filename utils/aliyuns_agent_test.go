package utils

import (
	"fmt"
	"testing"
)

func TestAgentScore(t *testing.T) {

	answer := "https://shijuan.obs.cn-east-3.myhuaweicloud.com/english/math1as.jpg"
	question := "https://shijuan.obs.cn-east-3.myhuaweicloud.com/english/shuxue111.jpg"

	//

	result, err := AgentMathScore(question, answer)
	if err != nil {
		t.Errorf("Error: %v", err)
	}
	fmt.Println(result.Text)
	fmt.Println(result.FinishReason)
	fmt.Println(result.RequestID)
	fmt.Println(result.SessionID)
}
