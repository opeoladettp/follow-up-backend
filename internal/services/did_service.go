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

type DIDService struct {
	apiKey     string
	httpClient *http.Client
}

type DIDTalkRequest struct {
	SourceURL string    `json:"source_url"`
	Script    DIDScript `json:"script"`
	Config    DIDConfig `json:"config,omitempty"`
}

type DIDScript struct {
	Type     string       `json:"type"`
	Input    string       `json:"input,omitempty"`   // for text type
	AudioURL string       `json:"audio_url,omitempty"` // for audio type (pre-generated TTS)
	Provider *DIDProvider `json:"provider,omitempty"`
}

type DIDProvider struct {
	Type    string `json:"type"`
	VoiceID string `json:"voice_id,omitempty"`
}

type DIDConfig struct {
	Fluent      bool   `json:"fluent,omitempty"`
	PadAudio    string `json:"pad_audio,omitempty"`
	StitchImage bool   `json:"stitch,omitempty"`
	// Note: reduce_noise and crop_type are supported on paid plans
}

type DIDTalkResponse struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	ResultURL string    `json:"result_url,omitempty"`
	VideoURL  string    `json:"video_url,omitempty"`  // some D-ID responses use video_url
	CreatedAt time.Time `json:"created_at"`
	Error     *struct {
		Kind        string `json:"kind"`
		Description string `json:"description"`
	} `json:"error,omitempty"`
}

func NewDIDService(apiKey string) *DIDService {
	return &DIDService{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GenerateVideo creates a talking avatar video using D-ID API.
// If audioURL is non-empty, it uses pre-generated audio (e.g. ElevenLabs cloned voice).
// Otherwise it uses D-ID's built-in TTS with the given voiceID.
func (d *DIDService) GenerateVideo(script, avatarURL, voiceID, audioURL string) (string, error) {
	if d.apiKey == "" {
		return "", fmt.Errorf("D-ID API key not configured")
	}

	url := "https://api.d-id.com/talks"

	request := DIDTalkRequest{
		SourceURL: avatarURL,
		Config: DIDConfig{
			Fluent:      true,
			PadAudio:    "0.0",
			StitchImage: true,
		},
	}

	if audioURL != "" {
		// Use pre-generated audio (cloned voice from ElevenLabs)
		request.Script = DIDScript{
			Type:     "audio",
			AudioURL: audioURL,
		}
	} else {
		// Use D-ID built-in TTS
		request.Script = DIDScript{
			Type:  "text",
			Input: script,
		}
		if voiceID != "" {
			request.Script.Provider = &DIDProvider{
				Type:    "microsoft",
				VoiceID: voiceID,
			}
		}
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"avatar_url":    avatarURL,
		"script_length": len(script),
		"voice_id":      voiceID,
		"request_body":  string(jsonData),
	}).Info("Sending request to D-ID API")

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", d.apiKey))
	req.Header.Set("Content-Type", "application/json")

	startTime := time.Now()
	resp, err := d.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"status_code":  resp.StatusCode,
		"response":     string(body),
		"latency_ms":   time.Since(startTime).Milliseconds(),
	}).Info("D-ID API response received")

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("D-ID API error (status %d): %s", resp.StatusCode, string(body))
	}

	var didResp DIDTalkResponse
	if err := json.Unmarshal(body, &didResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"talk_id": didResp.ID,
		"status":  didResp.Status,
	}).Info("D-ID video generation started")

	return didResp.ID, nil
}

// GetVideoStatus checks the status of a D-ID video generation
func (d *DIDService) GetVideoStatus(talkID string) (*DIDTalkResponse, error) {
	if d.apiKey == "" {
		return nil, fmt.Errorf("D-ID API key not configured")
	}

	url := fmt.Sprintf("https://api.d-id.com/talks/%s", talkID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", d.apiKey))

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("D-ID API error: %s", string(body))
	}

	var didResp DIDTalkResponse
	if err := json.Unmarshal(body, &didResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"talk_id":    talkID,
		"status":     didResp.Status,
		"result_url": didResp.ResultURL,
	}).Info("D-ID video status polled")

	return &didResp, nil
}

// CloneVoice creates a custom voice from audio samples (requires D-ID Pro plan)
func (d *DIDService) CloneVoice(name string, audioFiles []string) (string, error) {
	// Note: Voice cloning requires D-ID Pro plan or higher
	// This is a placeholder for the voice cloning endpoint
	// Actual implementation would upload audio files and create voice profile
	
	logrus.Warn("Voice cloning requires D-ID Pro plan - using default voice")
	return "", fmt.Errorf("voice cloning not implemented - requires Pro plan")
}

// ListVoices retrieves available voices from D-ID
func (d *DIDService) ListVoices() ([]map[string]interface{}, error) {
	if d.apiKey == "" {
		return nil, fmt.Errorf("D-ID API key not configured")
	}

	url := "https://api.d-id.com/tts/voices"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", d.apiKey))

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("D-ID API error: %s", string(body))
	}

	var voices []map[string]interface{}
	if err := json.Unmarshal(body, &voices); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return voices, nil
}
