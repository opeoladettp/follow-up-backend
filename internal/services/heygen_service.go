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

const (
	heygenBaseURL = "https://api.heygen.com"
	// Fallback stock avatar — "Jared Headshot" (professional, news-friendly)
	fallbackAvatarID = "906e3c1914a441bea7c8d0b1bebbc981"
	// Fallback English voice
	fallbackVoiceID = "f38a635bee7a4d1f9b0a654a31d050d2"
)

type HeyGenService struct {
	apiKey   string
	avatarID string // custom avatar ID (from HEYGEN_AVATAR_ID env var)
	voiceID  string // custom/cloned voice ID (from HEYGEN_VOICE_ID env var)
	client   *http.Client
}

type heygenVideoRequest struct {
	VideoInputs []heygenVideoInput `json:"video_inputs"`
	Dimension   heygenDimension    `json:"dimension"`
	Caption     bool               `json:"caption"`
}

type heygenVideoInput struct {
	Character heygenCharacter `json:"character"`
	Voice     heygenVoice     `json:"voice"`
}

type heygenCharacter struct {
	Type       string `json:"type"`
	AvatarID   string `json:"avatar_id"`
	AvatarStyle string `json:"avatar_style"`
}

type heygenVoice struct {
	Type    string `json:"type"`
	VoiceID string `json:"voice_id"`
	InputText string `json:"input_text"`
	Speed   float64 `json:"speed"`
}

type heygenDimension struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type heygenCreateResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		VideoID string `json:"video_id"`
	} `json:"data"`
}

type heygenStatusResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		VideoID   string `json:"video_id"`
		Status    string `json:"status"`
		VideoURL  string `json:"video_url"`
		Error     *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	} `json:"data"`
}

func NewHeyGenService(apiKey, avatarID, voiceID string) *HeyGenService {
	return &HeyGenService{
		apiKey:   apiKey,
		avatarID: avatarID,
		voiceID:  voiceID,
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

// WithOverrides returns a shallow copy of the service with per-request avatar/voice overrides.
// Empty strings leave the original value unchanged.
func (h *HeyGenService) WithOverrides(avatarID, voiceID string) *HeyGenService {
	copy := *h
	if avatarID != "" {
		copy.avatarID = avatarID
	}
	if voiceID != "" {
		copy.voiceID = voiceID
	}
	return &copy
}

// GenerateVideo submits a video generation job and returns the video_id.
// Falls back to built-in defaults if avatar/voice IDs are not configured.
func (h *HeyGenService) GenerateVideo(script string) (string, error) {
	avatarID := h.avatarID
	if avatarID == "" {
		avatarID = fallbackAvatarID
	}
	voiceID := h.voiceID
	if voiceID == "" {
		voiceID = fallbackVoiceID
	}

	// HeyGen has a 5000 char limit per input_text; truncate gracefully
	if len(script) > 4900 {
		script = script[:4900] + "..."
	}

	payload := heygenVideoRequest{
		VideoInputs: []heygenVideoInput{
			{
				Character: heygenCharacter{
					Type:        "avatar",
					AvatarID:    avatarID,
					AvatarStyle: "normal",
				},
				Voice: heygenVoice{
					Type:      "text",
					VoiceID:   voiceID,
					InputText: script,
					Speed:     1.0,
				},
			},
		},
		Dimension: heygenDimension{Width: 1280, Height: 720},
		Caption:   false,
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", heygenBaseURL+"/v2/video/generate", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("X-Api-Key", h.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("heygen request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result heygenCreateResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse heygen response: %w", err)
	}

	if result.Code != 100 {
		return "", fmt.Errorf("heygen error %d: %s", result.Code, result.Message)
	}

	logrus.WithField("video_id", result.Data.VideoID).Info("HeyGen video job submitted")
	return result.Data.VideoID, nil
}

// GetVideoStatus polls for the status of a video job.
func (h *HeyGenService) GetVideoStatus(videoID string) (status, videoURL string, err error) {
	req, err := http.NewRequest("GET", heygenBaseURL+"/v1/video_status.get?video_id="+videoID, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to create status request: %w", err)
	}
	req.Header.Set("X-Api-Key", h.apiKey)

	resp, err := h.client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("heygen status request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result heygenStatusResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", "", fmt.Errorf("failed to parse heygen status: %w", err)
	}

	if result.Code != 100 {
		return "", "", fmt.Errorf("heygen status error %d: %s", result.Code, result.Message)
	}

	if result.Data.Error != nil {
		return "failed", "", fmt.Errorf("heygen video failed: %s", result.Data.Error.Message)
	}

	return result.Data.Status, result.Data.VideoURL, nil
}

// WaitForVideo polls until the video is completed or failed (max 10 min).
func (h *HeyGenService) WaitForVideo(videoID string) (string, error) {
	deadline := time.Now().Add(10 * time.Minute)
	for time.Now().Before(deadline) {
		status, videoURL, err := h.GetVideoStatus(videoID)
		if err != nil {
			return "", err
		}
		logrus.WithFields(logrus.Fields{"video_id": videoID, "status": status}).Debug("HeyGen poll")
		switch status {
		case "completed":
			return videoURL, nil
		case "failed":
			return "", fmt.Errorf("heygen video generation failed")
		}
		time.Sleep(10 * time.Second)
	}
	return "", fmt.Errorf("heygen video generation timed out after 10 minutes")
}
