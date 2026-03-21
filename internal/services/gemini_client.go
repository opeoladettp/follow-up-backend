package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// GeminiClient handles communication with Google's Gemini API
type GeminiClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// GeminiRequest represents a request to the Gemini API
type GeminiRequest struct {
	Contents []GeminiContent `json:"contents"`
	GenerationConfig *GenerationConfig `json:"generationConfig,omitempty"`
}

type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
	Role  string       `json:"role,omitempty"`
}

type GeminiPart struct {
	Text string `json:"text"`
}

type GenerationConfig struct {
	Temperature     float64 `json:"temperature,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
	TopP            float64 `json:"topP,omitempty"`
	TopK            int     `json:"topK,omitempty"`
}

// GeminiResponse represents a response from the Gemini API
type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
}

// NewGeminiClient creates a new Gemini API client
func NewGeminiClient(apiKey string) *GeminiClient {
	return &GeminiClient{
		apiKey:  apiKey,
		baseURL: "https://generativelanguage.googleapis.com/v1beta",
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// GenerateContent sends a prompt to Gemini and returns the generated text with retry logic
func (g *GeminiClient) GenerateContent(prompt string) (string, error) {
	maxRetries := 3
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		result, err := g.generateContentOnce(prompt)
		if err == nil {
			return result, nil
		}

		lastErr = err
		if attempt < maxRetries {
			// Exponential backoff: 2s, 4s, 8s
			backoffDuration := time.Duration(1<<uint(attempt)) * time.Second
			logrus.WithFields(logrus.Fields{
				"attempt":  attempt,
				"max_retries": maxRetries,
				"backoff_ms": backoffDuration.Milliseconds(),
				"error": err.Error(),
			}).Warn("Gemini API call failed, retrying...")
			time.Sleep(backoffDuration)
		}
	}

	return "", fmt.Errorf("gemini API failed after %d attempts: %w", maxRetries, lastErr)
}

// generateContentOnce sends a single request to Gemini API
func (g *GeminiClient) generateContentOnce(prompt string) (string, error) {
	// Use Gemini 2.5 Flash for faster responses and higher rate limits
	url := fmt.Sprintf("%s/models/gemini-2.5-flash:generateContent?key=%s", g.baseURL, g.apiKey)

	request := GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{Text: prompt},
				},
			},
		},
		GenerationConfig: &GenerationConfig{
			Temperature:     0.7,
			MaxOutputTokens: 2048,
			TopP:            0.95,
			TopK:            40,
		},
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	startTime := time.Now()
	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logrus.WithFields(logrus.Fields{
			"status_code": resp.StatusCode,
			"response":    string(body),
		}).Error("Gemini API error")
		return "", fmt.Errorf("gemini API returned status %d: %s", resp.StatusCode, string(body))
	}

	var geminiResp GeminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content generated")
	}

	generatedText := geminiResp.Candidates[0].Content.Parts[0].Text

	logrus.WithFields(logrus.Fields{
		"latency_ms":    time.Since(startTime).Milliseconds(),
		"output_length": len(generatedText),
	}).Info("Gemini API call successful")

	return generatedText, nil
}

// GenerateWithGrounding generates content with web search grounding
func (g *GeminiClient) GenerateWithGrounding(prompt string) (string, error) {
	// Note: Grounding/web search requires Google AI Studio or Vertex AI
	// For now, we'll use the standard API and enhance the prompt
	enhancedPrompt := fmt.Sprintf(`%s

Please provide current, factual information. If you're unsure about recent events, indicate that clearly.`, prompt)

	return g.GenerateContent(enhancedPrompt)
}
