package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

type ElevenLabsService struct {
	apiKey     string
	httpClient *http.Client
}

type ElevenLabsVoice struct {
	VoiceID string `json:"voice_id"`
	Name    string `json:"name"`
}

func NewElevenLabsService(apiKey string) *ElevenLabsService {
	return &ElevenLabsService{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// CloneVoice uploads an audio sample and creates a cloned voice, returning the voice ID.
func (e *ElevenLabsService) CloneVoice(name string, audioData []byte, audioFilename string) (string, error) {
	if e.apiKey == "" {
		return "", fmt.Errorf("ElevenLabs API key not configured")
	}

	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	_ = w.WriteField("name", name)
	_ = w.WriteField("description", "Cloned voice for news reporter")
	_ = w.WriteField("remove_background_noise", "true")

	part, err := w.CreateFormFile("files", audioFilename)
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := part.Write(audioData); err != nil {
		return "", fmt.Errorf("failed to write audio data: %w", err)
	}
	w.Close()

	req, err := http.NewRequest("POST", "https://api.elevenlabs.io/v1/voices/add", &body)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("xi-api-key", e.apiKey)
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ElevenLabs API error (%d): %s", resp.StatusCode, string(respBody))
	}

	var result ElevenLabsVoice
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"voice_id": result.VoiceID,
		"name":     name,
	}).Info("ElevenLabs voice cloned successfully")

	return result.VoiceID, nil
}

// GenerateSpeech generates TTS audio from text using a voice ID, returns audio bytes (mp3).
func (e *ElevenLabsService) GenerateSpeech(text, voiceID string) ([]byte, error) {
	if e.apiKey == "" {
		return nil, fmt.Errorf("ElevenLabs API key not configured")
	}

	url := fmt.Sprintf("https://api.elevenlabs.io/v1/text-to-speech/%s", voiceID)

	payload := map[string]interface{}{
		"text":     text,
		"model_id": "eleven_turbo_v2_5",
		"voice_settings": map[string]interface{}{
			"stability":        0.4,
			"similarity_boost": 0.85,
			"style":            0.3,
			"use_speaker_boost": true,
		},
	}

	jsonData, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("xi-api-key", e.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "audio/mpeg")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ElevenLabs TTS error (%d): %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}
