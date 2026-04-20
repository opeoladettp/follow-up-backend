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
	// Fallback stock avatar
	fallbackAvatarID = "Jared_sitting_sofa_20220818"
	// Fallback English voice
	fallbackVoiceID = "1bd001e7e50f421d891986aad5158bc8"
)

type HeyGenService struct {
	apiKey   string
	avatarID string
	voiceID  string
	client   *http.Client
}

// heygenV2Request matches POST /v2/videos.
// callback_id is echoed back in the webhook payload so we can match the video to a report.
type heygenV2Request struct {
	AvatarID    string             `json:"avatar_id"`
	Script      string             `json:"script"`
	VoiceID     string             `json:"voice_id"`
	CallbackID  string             `json:"callback_id,omitempty"`
	Title       string             `json:"title,omitempty"`
	AspectRatio string             `json:"aspect_ratio,omitempty"`
	Voice       *heygenVoiceTuning `json:"voice,omitempty"`
}

type heygenVoiceTuning struct {
	Speed float64 `json:"speed,omitempty"`
}

type heygenV2Response struct {
	VideoID string `json:"video_id"`
	Status  string `json:"status"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type heygenV2StatusResponse struct {
	VideoID  string  `json:"video_id"`
	Status   string  `json:"status"`
	VideoURL string  `json:"video_url"`
	Duration float64 `json:"duration"`
	Error    *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	// Legacy wrapper (some endpoints still wrap in data{})
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    *struct {
		VideoID  string `json:"video_id"`
		Status   string `json:"status"`
		VideoURL string `json:"video_url"`
		Error    *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	} `json:"data"`
}

// HeyGenWebhookPayload is the POST body HeyGen sends to our webhook endpoint.
type HeyGenWebhookPayload struct {
	EventType string `json:"event_type"`
	EventData struct {
		VideoID    string `json:"video_id"`
		URL        string `json:"url"`
		Msg        string `json:"msg"`
		CallbackID string `json:"callback_id"` // we set this to the report_id
	} `json:"event_data"`
}

func NewHeyGenService(apiKey, avatarID, voiceID string) *HeyGenService {
	return &HeyGenService{
		apiKey:   apiKey,
		avatarID: avatarID,
		voiceID:  voiceID,
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

// WithOverrides returns a shallow copy with per-request avatar/voice overrides.
func (h *HeyGenService) WithOverrides(avatarID, voiceID string) *HeyGenService {
	cp := *h
	if avatarID != "" {
		cp.avatarID = avatarID
	}
	if voiceID != "" {
		cp.voiceID = voiceID
	}
	return &cp
}

// GenerateVideo submits a job to HeyGen and returns the video_id.
// reportID is passed as callback_id so the webhook can match the video back to the report.
func (h *HeyGenService) GenerateVideo(script, reportID string) (string, error) {
	avatarID := h.avatarID
	if avatarID == "" {
		avatarID = fallbackAvatarID
	}
	voiceID := h.voiceID
	if voiceID == "" {
		voiceID = fallbackVoiceID
	}

	if len(script) > 4900 {
		script = script[:4900] + "..."
	}

	payload := heygenV2Request{
		AvatarID:    avatarID,
		Script:      script,
		VoiceID:     voiceID,
		CallbackID:  reportID,
		AspectRatio: "16:9",
		Voice:       &heygenVoiceTuning{Speed: 1.0},
	}

	body, _ := json.Marshal(payload)

	logrus.WithFields(logrus.Fields{
		"avatar_id":   avatarID,
		"voice_id":    voiceID,
		"script_len":  len(script),
		"callback_id": reportID,
	}).Info("Submitting video to HeyGen")

	req, err := http.NewRequest("POST", heygenBaseURL+"/v2/videos", bytes.NewReader(body))
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

	logrus.WithFields(logrus.Fields{
		"http_status":   resp.StatusCode,
		"response_body": string(respBody),
	}).Info("HeyGen submit response")

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return "", fmt.Errorf("heygen auth failed (HTTP %d) — check HEYGEN_API_KEY", resp.StatusCode)
	}

	var result heygenV2Response
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse heygen response (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	if result.VideoID == "" {
		return "", fmt.Errorf("heygen error (HTTP %d, code %d): %s — body: %s",
			resp.StatusCode, result.Code, result.Message, string(respBody))
	}

	logrus.WithFields(logrus.Fields{
		"video_id":    result.VideoID,
		"callback_id": reportID,
	}).Info("HeyGen video job submitted")
	return result.VideoID, nil
}

// GetVideoStatus fetches the current status of a video job.
func (h *HeyGenService) GetVideoStatus(videoID string) (status, videoURL string, err error) {
	req, err := http.NewRequest("GET", heygenBaseURL+"/v2/videos/"+videoID, nil)
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

	logrus.WithFields(logrus.Fields{
		"video_id":      videoID,
		"http_status":   resp.StatusCode,
		"response_body": string(respBody),
	}).Debug("HeyGen status response")

	var result heygenV2StatusResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", "", fmt.Errorf("failed to parse heygen status (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	s, u := result.Status, result.VideoURL
	var vidErr *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	vidErr = result.Error

	if result.Data != nil {
		if result.Data.Status != "" {
			s = result.Data.Status
		}
		if result.Data.VideoURL != "" {
			u = result.Data.VideoURL
		}
		if result.Data.Error != nil {
			vidErr = result.Data.Error
		}
	}

	if vidErr != nil {
		return "failed", "", fmt.Errorf("heygen video failed: %s", vidErr.Message)
	}

	return s, u, nil
}

// RegisterWebhook registers our callback URL with HeyGen for avatar_video events.
// Safe to call on startup — it upserts if the URL is already registered.
func (h *HeyGenService) RegisterWebhook(callbackURL string) error {
	// First list existing webhooks to avoid duplicates
	existing, err := h.listWebhooks()
	if err == nil {
		for _, ep := range existing {
			if ep.URL == callbackURL {
				logrus.WithField("url", callbackURL).Info("HeyGen webhook already registered")
				return nil
			}
		}
	}

	payload := map[string]interface{}{
		"url":    callbackURL,
		"events": []string{"avatar_video.success", "avatar_video.fail"},
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", heygenBaseURL+"/v1/webhook/endpoint.add", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("X-Api-Key", h.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to register webhook: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	logrus.WithFields(logrus.Fields{
		"url":           callbackURL,
		"http_status":   resp.StatusCode,
		"response_body": string(respBody),
	}).Info("HeyGen webhook registration response")

	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook registration failed (HTTP %d): %s", resp.StatusCode, string(respBody))
	}
	return nil
}

type heygenWebhookEndpoint struct {
	EndpointID string   `json:"endpoint_id"`
	URL        string   `json:"url"`
	Events     []string `json:"events"`
}

func (h *HeyGenService) listWebhooks() ([]heygenWebhookEndpoint, error) {
	req, err := http.NewRequest("GET", heygenBaseURL+"/v1/webhook/endpoint.list", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Api-Key", h.apiKey)

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			Endpoints []heygenWebhookEndpoint `json:"endpoints"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Data.Endpoints, nil
}

// WaitForVideo is kept for local/dev use where webhooks aren't reachable.
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
