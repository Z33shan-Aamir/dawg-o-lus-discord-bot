package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/discord-bitch/config"
)

type Inference struct {
	apiToken string
	model    string
	baseURL  string
	// systemPrompt            string
	responseCriteria        string
	isResponseRequiredModel string
}

func New(cfg *config.Config) *Inference {
	return &Inference{
		apiToken: cfg.AiAPIKey,
		model:    cfg.ModelName,
		baseURL:  "https://openrouter.ai/api/v1",
		// systemPrompt:            cfg.SystemPrompt,
		responseCriteria:        cfg.ResponseCriteria,
		isResponseRequiredModel: cfg.IsResponseRequiredModel,
	}
}

func NewOllama(cfg *config.Config) *Inference {
	return &Inference{
		apiToken: "",
		model:    "meta-llama/llama-3.1-8b-instruct",
		baseURL:  "http://localhost:11434/v1",
		// systemPrompt:     cfg.SystemPrompt,
		responseCriteria: cfg.ResponseCriteria,
	}

}

func (i *Inference) chatCompletions(systemPrompt string, model string, history []Message, newMessage string) (string, error) {
	messages := []map[string]string{
		{"role": "system", "content": systemPrompt},
	}

	for _, msg := range history {
		messages = append(messages, map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}

	messages = append(messages, map[string]string{
		"role":    "user",
		"content": newMessage,
	})

	if os.Getenv("IS_PROD") == "false" {
		log.Info("message is", newMessage)
	}

	body := map[string]any{
		"model":    model,
		"messages": messages,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal error: %w", err)
	}

	req, err := http.NewRequest("POST", i.baseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("request creation error: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if i.apiToken != "" {
		req.Header.Set("Authorization", "Bearer "+i.apiToken)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("api returned status code %d", resp.StatusCode)
	}

	// Structural tag explicitly fixed here (`json:"message"`)
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode error: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("empty choices returned from model")
	}

	if os.Getenv("IS_PROD") == "false" {
		log.Info("message is", result.Choices[0].Message.Content)
	}
	return result.Choices[0].Message.Content, nil
}

func (i *Inference) IsResponseRequired(message string, history []Message) (bool, error) {
	result, err := i.chatCompletions(i.responseCriteria, i.isResponseRequiredModel, history, message)
	if err != nil {
		log.Error("error evaluating if response is required", "err", err)
		return false, err
	}

	if strings.ToLower(strings.TrimSpace(result)) == "no" {
		return false, nil
	}

	return true, nil
}

func (i *Inference) GenerateResponse(systemPrompt string, history []Message, newMessage string) (string, error) {

	message, err := i.chatCompletions(systemPrompt, i.model, history, newMessage)
	if err != nil {
		log.Error("Could not generate response", err)
		return "", err
	}

	return message, nil
}
func (i *Inference) ModelName() string {
	return i.model
}
