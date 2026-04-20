package services

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"followupmedium-newsroom/internal/models"

	"github.com/sirupsen/logrus"
)

type AIService struct {
	apiKey           string
	geminiClient     *GeminiClient
	newsAPIKey       string
	didService       *DIDService
	heygenService    *HeyGenService
	elevenLabsSvc    *ElevenLabsService
	googleImageSvc   *GoogleImageService
	imagenService    *GoogleImagenService
	s3Service        *S3Service
}

func NewAIService(apiKey, newsAPIKey string) *AIService {
	geminiClient := NewGeminiClient(apiKey)
	return &AIService{
		apiKey:           apiKey,
		geminiClient:     geminiClient,
		newsAPIKey:       newsAPIKey,
		didService:       nil, // Will be set when DID_API_KEY is configured
		googleImageSvc:   NewGoogleImageService(geminiClient), // Initialize with Gemini client
		imagenService:    nil, // Will be set when Google Cloud project is configured
		s3Service:        nil, // Will be set when AWS credentials are configured
	}
}

// SetDIDService configures the D-ID service for video generation
func (a *AIService) SetDIDService(didAPIKey string) {
	if didAPIKey != "" {
		a.didService = NewDIDService(didAPIKey)
		logrus.Info("D-ID service configured for video generation")
	}
}

// SetHeyGenService configures the HeyGen service for video generation
func (a *AIService) SetHeyGenService(apiKey, avatarID, voiceID string) {
	if apiKey != "" {
		a.heygenService = NewHeyGenService(apiKey, avatarID, voiceID)
		logrus.WithFields(logrus.Fields{
			"avatar_id_set": avatarID != "",
			"voice_id_set":  voiceID != "",
		}).Info("HeyGen service configured for video generation")
	}
}

// SetElevenLabsService configures the ElevenLabs service for voice cloning
func (a *AIService) SetElevenLabsService(apiKey string) {
	if apiKey != "" {
		a.elevenLabsSvc = NewElevenLabsService(apiKey)
		logrus.Info("ElevenLabs service configured for voice cloning")
	}
}

// SetImagenService configures the Google Imagen service for image generation
func (a *AIService) SetImagenService(svc *GoogleImagenService) {
	a.imagenService = svc
	logrus.Info("Google Imagen service configured for image generation")
}

// SetS3Service configures the S3 service for image storage
func (a *AIService) SetS3Service(svc *S3Service) {
	a.s3Service = svc
	logrus.Info("S3 service configured for image storage")
}

// UploadAvatarToS3 uploads a base64 data URL or remote URL avatar to S3 and returns
// a presigned URL valid for 1 hour. D-ID requires a publicly accessible HTTPS URL.
// didValidImageURL returns true if the URL ends with a D-ID accepted extension.
func didValidImageURL(u string) bool {
	lower := strings.ToLower(u)
	// Strip query string before checking extension
	if idx := strings.Index(lower, "?"); idx != -1 {
		lower = lower[:idx]
	}
	return strings.HasSuffix(lower, ".jpg") ||
		strings.HasSuffix(lower, ".jpeg") ||
		strings.HasSuffix(lower, ".png")
}

// UploadAvatarToS3 ensures the avatar URL is a D-ID-compatible public image URL.
// It handles three cases:
//  1. data: URLs (base64) → decode and upload to S3
//  2. Remote URLs already ending in .jpg/.jpeg/.png → use as-is
//  3. Remote URLs without valid extension (e.g. Google profile pics) → download and re-upload to S3 as .jpg
// PrepareAvatarURL returns the avatar URL ready for the video provider.
// For HeyGen, the avatar URL is not used (HeyGen uses stock avatars), so we return as-is.
// For D-ID, we upload to S3 to get a clean public HTTPS URL.
func (a *AIService) PrepareAvatarURL(avatarURL, reportID string) (string, error) {
	if a.heygenService != nil {
		// HeyGen uses its own stock avatars — no need to process the user's avatar
		return avatarURL, nil
	}
	return a.UploadAvatarToS3(avatarURL, reportID)
}

func (a *AIService) UploadAvatarToS3(avatarURL, reportID string) (string, error) {
	if a.s3Service == nil {
		return avatarURL, nil
	}

	// Case 1: base64 data URL
	if strings.HasPrefix(avatarURL, "data:") {
		comma := strings.Index(avatarURL, ",")
		if comma == -1 {
			return "", fmt.Errorf("invalid data URL format")
		}
		meta := avatarURL[5:comma]
		b64data := avatarURL[comma+1:]

		contentType := "image/jpeg"
		ext := "jpg"
		if strings.Contains(meta, "png") {
			contentType = "image/png"
			ext = "png"
		} else if strings.Contains(meta, "webp") {
			contentType = "image/webp"
			ext = "webp"
		}

		imgData, err := base64.StdEncoding.DecodeString(b64data)
		if err != nil {
			imgData, err = base64.URLEncoding.DecodeString(b64data)
			if err != nil {
				return "", fmt.Errorf("failed to decode avatar image: %w", err)
			}
		}

		key := fmt.Sprintf("avatars/%s/avatar.%s", reportID, ext)
		s3URL, err := a.s3Service.UploadImage(key, imgData, contentType)
		if err != nil {
			return "", fmt.Errorf("failed to upload avatar to S3: %w", err)
		}
		logrus.WithFields(logrus.Fields{"report_id": reportID, "s3_url": s3URL}).Info("Avatar (base64) uploaded to S3")
		return s3URL, nil
	}

	// Case 2: already a valid D-ID image URL
	if didValidImageURL(avatarURL) {
		logrus.WithField("avatar_url", avatarURL).Info("Avatar URL already valid for D-ID, using as-is")
		return avatarURL, nil
	}

	// Case 3: remote URL without valid extension (e.g. Google profile picture)
	logrus.WithFields(logrus.Fields{
		"report_id":  reportID,
		"avatar_url": avatarURL,
	}).Info("Avatar URL lacks valid extension, downloading and re-uploading to S3")

	resp, err := http.Get(avatarURL) //nolint:gosec
	if err != nil {
		return "", fmt.Errorf("failed to download avatar image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download avatar image: HTTP %d", resp.StatusCode)
	}

	imgData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read avatar image body: %w", err)
	}

	// Detect content type from response or sniff from bytes
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = http.DetectContentType(imgData)
	}
	// Normalise to just the mime type (strip params like "; charset=...")
	if idx := strings.Index(contentType, ";"); idx != -1 {
		contentType = strings.TrimSpace(contentType[:idx])
	}

	ext := "jpg"
	if contentType == "image/png" {
		ext = "png"
	}

	// If it's a webp, convert by re-encoding isn't trivial - just store as jpg key
	// D-ID accepts jpg so we always use jpg extension regardless
	key := fmt.Sprintf("avatars/%s/avatar.%s", reportID, ext)
	s3URL, err := a.s3Service.UploadImage(key, imgData, "image/jpeg")
	if err != nil {
		return "", fmt.Errorf("failed to upload avatar to S3: %w", err)
	}

	logrus.WithFields(logrus.Fields{"report_id": reportID, "s3_url": s3URL}).Info("Avatar (remote) uploaded to S3")
	return s3URL, nil
}

// stripHTML removes all HTML tags and decodes HTML entities from a string.
// This is used to clean RSS feed descriptions before passing to Gemini.
func stripHTML(s string) string {
	// Remove <style>...</style> and <script>...</script> blocks entirely
	re := regexp.MustCompile(`(?is)<(style|script)[^>]*>.*?</(style|script)>`)
	s = re.ReplaceAllString(s, " ")

	// Remove all remaining HTML tags
	re = regexp.MustCompile(`<[^>]+>`)
	s = re.ReplaceAllString(s, " ")

	// Decode common HTML entities
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", `"`)
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&apos;", "'")

	// Collapse multiple spaces/newlines
	re = regexp.MustCompile(`[ \t]{2,}`)
	s = re.ReplaceAllString(s, " ")
	re = regexp.MustCompile(`\n{3,}`)
	s = re.ReplaceAllString(s, "\n\n")

	return strings.TrimSpace(s)
}

// stripImagePrompts removes image prompt sections from the script
func (a *AIService) stripImagePrompts(script string) string {
	// Remove [IMAGE PROMPTS FOR GENERATION] section and everything until the next major section
	re := regexp.MustCompile(`(?i)\[IMAGE\s+PROMPTS\s+FOR\s+GENERATION\].*?(?:---|\n\n[A-Z]|$)`)
	script = re.ReplaceAllString(script, "")
	
	// Remove Story Image Prompt: lines
	re = regexp.MustCompile(`(?i)Story\s+Image\s+Prompt:.*?(?:\n\n|Reaction|Background|$)`)
	script = re.ReplaceAllString(script, "")
	
	// Remove Reaction Image Prompt: lines
	re = regexp.MustCompile(`(?i)Reaction\s+Image\s+Prompt:.*?(?:\n\n|Background|$)`)
	script = re.ReplaceAllString(script, "")
	
	// Remove Background Image Prompt: lines
	re = regexp.MustCompile(`(?i)Background\s+Image\s+Prompt:.*?(?:\n\n|$)`)
	script = re.ReplaceAllString(script, "")
	
	// Clean up extra whitespace
	script = strings.TrimSpace(script)
	
	return script
}

// stripMarkdown removes markdown formatting from text
func (a *AIService) stripMarkdown(text string) string {
	// Remove markdown headers (##, ###, etc.)
	re := regexp.MustCompile(`(?m)^#+\s+`)
	text = re.ReplaceAllString(text, "")
	
	// Remove bold (**text** or __text__)
	re = regexp.MustCompile(`\*\*(.+?)\*\*`)
	text = re.ReplaceAllString(text, "$1")
	re = regexp.MustCompile(`__(.+?)__`)
	text = re.ReplaceAllString(text, "$1")
	
	// Remove italic (*text* or _text_)
	re = regexp.MustCompile(`\*(.+?)\*`)
	text = re.ReplaceAllString(text, "$1")
	re = regexp.MustCompile(`_(.+?)_`)
	text = re.ReplaceAllString(text, "$1")
	
	// Remove code blocks (```...```)
	re = regexp.MustCompile("(?s)```.*?```")
	text = re.ReplaceAllString(text, "")
	
	// Remove inline code (`text`)
	re = regexp.MustCompile("`(.+?)`")
	text = re.ReplaceAllString(text, "$1")
	
	// Remove links [text](url)
	re = regexp.MustCompile(`\[(.+?)\]\(.+?\)`)
	text = re.ReplaceAllString(text, "$1")
	
	// Remove horizontal rules (---, ***, ___)
	re = regexp.MustCompile(`(?m)^[-*_]{3,}$`)
	text = re.ReplaceAllString(text, "")
	
	// Clean up extra whitespace
	text = strings.TrimSpace(text)
	
	return text
}

// GenerateNewsReport generates a comprehensive news report script from a headline using Gemini API
func (a *AIService) GenerateNewsReport(title, description, url, authorName string) (string, error) {
	startTime := time.Now()

	// Sanitize inputs — RSS descriptions often contain raw HTML tags and entities
	title = stripHTML(title)
	description = stripHTML(description)

	correspondentName := authorName
	if correspondentName == "" {
		correspondentName = "your correspondent"
	}

	isTwitterSource := strings.Contains(url, "x.com") || strings.Contains(url, "twitter.com")

	var researchInstruction string
	if isTwitterSource {
		researchInstruction = `This story originated from a social media post. You MUST:
- Expand far beyond the tweet using your full knowledge of the topic, key figures, and context
- Draw on what you know about reactions from prominent voices, commentators, and institutions
- Reference relevant historical context, previous statements, or related events
- Treat this as a full investigative news report, not a social media summary`
	} else {
		researchInstruction = `This story is from a news source. You MUST:
- Expand significantly beyond the provided description using your full knowledge
- Provide deep background on the key figures, institutions, and events mentioned
- Draw on related developments, historical context, and expert perspectives
- Reference comparable events or precedents where relevant`
	}

	prompt := fmt.Sprintf(`You are a senior broadcast journalist named %s at a world-class international news outlet like BBC World Service or CNN International.

HEADLINE: %s
STORY DETAILS: %s

%s

MANDATORY REQUIREMENTS — EVERY SINGLE ONE MUST BE MET:

LENGTH: Write AT LEAST 8 full paragraphs. Each paragraph must be 3 to 5 sentences. This is non-negotiable. Do not stop early. Do not summarise. Write every paragraph in full.

STRUCTURE:
Paragraph 1 — Opening: Varied, natural intro matching the story tone. Choose from: "Good evening", "Good morning", "In a significant development", "Turning now to", "A major story is developing" — never default to "breaking news" unless truly urgent.
Paragraph 2 — Core facts: Who, what, when, where, why — clearly and fully stated.
Paragraph 3 — Key figures: Expand on who the people and institutions involved are, their roles and significance.
Paragraph 4 — Background: What led to this, historical significance, relevant precedents.
Paragraph 5 — Wider significance: Why this matters nationally and internationally.
Paragraph 6 — Reactions: How different groups, communities, and institutions are responding.
Paragraph 7 — Analysis: What observers, analysts, and commentators are noting about implications.
Paragraph 8 — What next: Upcoming developments, unanswered questions, what to watch.
Closing paragraph — Sign-off: "Reporting for [outlet], I'm %s."

FORMAT — ABSOLUTE RULES:
- Plain text only. Zero asterisks, zero bold, zero bullet points, zero headers, zero markdown of any kind.
- Write exactly as spoken on air. No stage directions. No parenthetical notes.
- No fabricated direct quotes from named individuals.
- No mention of source URLs.

Write the complete, full-length script now. Do not truncate:`,
		correspondentName, title, description, researchInstruction, correspondentName)

	script, err := a.geminiClient.GenerateContent(prompt)
	if err != nil {
		logrus.WithError(err).Error("Failed to generate report with Gemini API")
		script = a.generateFallbackScript(title, description, "your correspondent", correspondentName)
	}

	script = a.stripImagePrompts(script)
	script = stripMarkdown(script)

	logrus.WithFields(logrus.Fields{
		"latency_ms":    time.Since(startTime).Milliseconds(),
		"title":         title,
		"author":        authorName,
		"script_length": len(script),
	}).Info("News report generated")

	return script, nil
}

// generateFallbackScript provides a fallback if Gemini API fails
func (a *AIService) generateFallbackScript(title, description, correspondentIntro, authorName string) string {
	reactions := `• Key figures are discussing the implications of this development
• Experts in the field are sharing their analysis and perspectives
• Public reaction has been significant, with widespread engagement
• Stakeholders are calling for careful consideration of all aspects`

	background := `This development comes amid ongoing attention to related issues. The situation has been evolving over time, with various factors contributing to the current state of affairs.`

	analysis := `What makes this particularly significant is the broader context and potential implications. Observers note that this could have lasting effects on how similar situations are handled in the future.`

	return fmt.Sprintf(`Good evening, %s, reporting live with breaking news.

%s

%s

%s

%s

The story is generating significant reactions. Here's what we're hearing:

%s

This development has far-reaching implications. Experts suggest this could fundamentally change how we approach this issue in the coming months and years.

Our team will continue monitoring this story as it develops. We'll bring you updates as soon as new information becomes available.

This is %s, and I'll be following this story closely. Stay with us for more updates.`, 
		correspondentIntro, title, description, background, analysis, reactions, authorName)
}

// Helper functions to simulate content (replace with actual Gemini API calls in production)
func generateSimulatedReactions(title string) string {
	// Generate context-appropriate reactions based on the title
	// In production, this will use actual X.com API and web search
	
	// For now, create generic but contextual reactions
	return fmt.Sprintf(`• Key figures on X are discussing the implications of this development
• Experts in the field are sharing their analysis and perspectives
• Public reaction has been significant, with thousands engaging in the conversation
• Stakeholders are calling for careful consideration of all aspects of this situation`)
}

func generateSimulatedBackground(title, description string) string {
	// Generate contextual background based on the story
	// In production, this will use Gemini with web search
	
	return fmt.Sprintf(`This development comes amid ongoing attention to related issues. The situation has been evolving over time, with various factors contributing to the current state of affairs.`)
}

func generateSimulatedAnalysis(title, description string) string {
	// Generate contextual analysis based on the story
	// In production, this will use Gemini deep thinking
	
	return fmt.Sprintf(`What makes this particularly significant is the broader context and potential implications. Observers note that this could have lasting effects on how similar situations are handled in the future.`)
}

// RefineNewsReport uses Gemini AI to refine the script based on user feedback
func (a *AIService) RefineNewsReport(currentScript, userFeedback, authorName string) (string, error) {
	startTime := time.Now()
	
	// Remove any previous refinement notes
	if idx := strings.Index(currentScript, "[Refined based on:"); idx != -1 {
		currentScript = currentScript[:idx]
		currentScript = strings.TrimSpace(currentScript)
	}
	
	// Create prompt for Gemini to refine the script
	prompt := fmt.Sprintf(`You are refining a news report script based on correspondent feedback.

CURRENT SCRIPT:
%s

CORRESPONDENT FEEDBACK:
%s

CORRESPONDENT NAME: %s

Please refine the script based on the feedback. Guidelines:
- If they ask for more content, expand relevant sections
- If they ask for reactions, add more detail about public/expert responses
- If they ask for background, add more context
- If they ask to make it more engaging, add dramatic elements and stronger language
- Maintain the professional news anchor style
- Keep the correspondent's name (%s) throughout
- Do NOT add notes or explanations, just return the refined script
- Do NOT include source URLs
- Do NOT include any image prompts or image generation instructions

Return ONLY the refined script:`, currentScript, userFeedback, authorName, authorName)

	// Call Gemini API
	refinedScript, err := a.geminiClient.GenerateContent(prompt)
	if err != nil {
		logrus.WithError(err).Error("Failed to refine report with Gemini API")
		// Fallback to keyword-based refinement
		return a.refineWithKeywords(currentScript, userFeedback, authorName), nil
	}

	// Strip any image prompts that might have been included
	refinedScript = a.stripImagePrompts(refinedScript)

	logrus.WithFields(logrus.Fields{
		"latency":  time.Since(startTime).Milliseconds(),
		"feedback": userFeedback,
		"author":   authorName,
		"method":   "gemini_api",
	}).Info("News report refined with Gemini")

	return refinedScript, nil
}

// refineWithKeywords provides fallback keyword-based refinement
func (a *AIService) refineWithKeywords(currentScript, userFeedback, authorName string) string {
	feedbackLower := strings.ToLower(userFeedback)
	
	if strings.Contains(feedbackLower, "too short") || strings.Contains(feedbackLower, "longer") {
		currentScript = expandScript(currentScript, authorName)
	}
	
	if strings.Contains(feedbackLower, "reaction") || strings.Contains(feedbackLower, "response") {
		currentScript = addSocialReactions(currentScript)
	}
	
	if strings.Contains(feedbackLower, "background") || strings.Contains(feedbackLower, "context") {
		currentScript = addBackgroundContext(currentScript)
	}
	
	if strings.Contains(feedbackLower, "engaging") || strings.Contains(feedbackLower, "dramatic") {
		currentScript = makeMoreEngaging(currentScript, authorName)
	}
	
	return currentScript
}

// Helper functions for script refinement
func expandScript(script, authorName string) string {
	// Add expert analysis section if not present
	if !strings.Contains(script, "expert") && !strings.Contains(script, "analysis") {
		expertSection := `

Expert Analysis:
Medical researchers emphasize that while this development is promising, it represents just one step in a longer journey. The technology behind this approach could pave the way for similar breakthroughs in other areas of preventive medicine.

Industry insiders suggest that if clinical trials prove successful, we could see this vaccine available to the public within the next 3-5 years, pending regulatory approval.`
		
		// Insert before the closing
		if idx := strings.LastIndex(script, "This is"); idx != -1 {
			script = script[:idx] + expertSection + "\n\n" + script[idx:]
		}
	}
	return script
}

func addSocialReactions(script string) string {
	// Add social media reactions if not present
	if !strings.Contains(script, "X are") && !strings.Contains(script, "social media") {
		reactionsSection := `

Social Media Reactions:
The announcement has sparked widespread discussion on X (formerly Twitter):

• Dr. Anthony Fauci (@NIAIDNews): "Exciting development in vaccine research. Universal vaccines have been a holy grail for decades."

• Bill Gates (@BillGates): "This kind of innovation is exactly what we need to prepare for future pandemics."

• Leading virologists are sharing the research paper, noting the innovative approach to targeting multiple virus families simultaneously.

• Public health advocates are calling for accelerated but thorough testing to ensure safety and efficacy.

The hashtag #UniversalVaccine is trending with over 50,000 mentions in the past hour.`
		
		// Insert before the closing
		if idx := strings.LastIndex(script, "This is"); idx != -1 {
			script = script[:idx] + reactionsSection + "\n\n" + script[idx:]
		}
	}
	return script
}

func addBackgroundContext(script string) string {
	// Add background context if not present
	if !strings.Contains(script, "background") && !strings.Contains(script, "history") {
		backgroundSection := `

Background Context:
This breakthrough builds on decades of research into universal vaccine technology. Previous attempts to create broad-spectrum vaccines have faced challenges in maintaining effectiveness across different virus strains.

The Stanford team's approach uses a novel mechanism that targets conserved regions found across multiple respiratory viruses, rather than the rapidly mutating surface proteins that traditional vaccines target.

Similar research is underway at institutions worldwide, but this study represents one of the most promising results to date.`
		
		// Insert after the main description
		lines := strings.Split(script, "\n")
		if len(lines) > 5 {
			// Insert after first few paragraphs
			insertPoint := 5
			newLines := append(lines[:insertPoint], append(strings.Split(backgroundSection, "\n"), lines[insertPoint:]...)...)
			script = strings.Join(newLines, "\n")
		}
	}
	return script
}

func makeMoreEngaging(script, authorName string) string {
	// Add dramatic opening if needed
	if !strings.Contains(script, "could change") && !strings.Contains(script, "revolutionary") {
		// Make the opening more dramatic
		script = strings.Replace(script, "Good evening,", "Good evening, and welcome to what could be a historic moment in medical science.", 1)
		
		// Add impact statement
		impactStatement := `

This could be the medical breakthrough we've been waiting for - a single vaccine that could end the annual cycle of flu seasons and reduce the burden of respiratory illnesses that affect millions worldwide.`
		
		// Insert early in the script
		lines := strings.Split(script, "\n")
		if len(lines) > 3 {
			insertPoint := 3
			newLines := append(lines[:insertPoint], append(strings.Split(impactStatement, "\n"), lines[insertPoint:]...)...)
			script = strings.Join(newLines, "\n")
		}
	}
	return script
}

// AnalyzeContent uses Gemini 3 Pro to analyze story developments
func (a *AIService) AnalyzeContent(content string, storyContext *models.StoryLifecycle) (*models.AIAnalysis, error) {
	startTime := time.Now()
	
	// TODO: Implement actual Gemini API call
	// For now, return mock analysis
	analysis := &models.AIAnalysis{
		Sentiment:   "neutral",
		Importance:  5,
		Summary:     "Content analysis pending Gemini API integration",
		KeyPoints:   []string{"Key point 1", "Key point 2"},
		GeneratedAt: time.Now(),
	}

	logrus.WithFields(logrus.Fields{
		"latency":    time.Since(startTime).Milliseconds(),
		"sentiment":  analysis.Sentiment,
		"importance": analysis.Importance,
	}).Info("Content analysis completed (mock)")

	return analysis, nil
}

// TriggerProductionPipeline generates video using D-ID API and uploads to S3
func (a *AIService) TriggerProductionPipeline(scriptText, identityImageURL, reportID, voiceAudioURL, heygenAvatarID, heygenVoiceID string) (string, error) {
	startTime := time.Now()

	// Prefer HeyGen (free trial, better quality)
	if a.heygenService != nil {
		logrus.WithField("report_id", reportID).Info("Using HeyGen for video generation")

		// Per-user IDs override the service-level defaults when provided
		svc := a.heygenService
		if heygenAvatarID != "" || heygenVoiceID != "" {
			svc = svc.WithOverrides(heygenAvatarID, heygenVoiceID)
		}

		videoID, err := svc.GenerateVideo(scriptText)
		if err != nil {
			logrus.WithError(err).Error("HeyGen video generation failed")
			return "", fmt.Errorf("HeyGen video generation failed: %w", err)
		}
		logrus.WithFields(logrus.Fields{
			"latency_ms": time.Since(startTime).Milliseconds(),
			"video_id":   videoID,
			"report_id":  reportID,
		}).Info("HeyGen video job submitted")
		return videoID, nil
	}

	// Fall back to D-ID if HeyGen not configured
	if a.didService == nil {
		logrus.Warn("No video service configured, using mock video generation")
		return a.mockVideoGeneration()
	}

	voiceID := "en-US-AriaNeural"
	script := scriptText
	if len(script) > 900 {
		script = script[:900]
		if idx := strings.LastIndexAny(script, ".!?"); idx > 0 {
			script = script[:idx+1]
		}
		logrus.WithField("original_length", len(scriptText)).Warn("Script truncated to fit D-ID limit")
	}

	logrus.WithFields(logrus.Fields{
		"avatar_url":    identityImageURL,
		"script_length": len(scriptText),
		"report_id":     reportID,
	}).Info("Calling D-ID API to generate video")

	talkID, err := a.didService.GenerateVideo(script, identityImageURL, voiceID, voiceAudioURL)
	if err != nil {
		return "", fmt.Errorf("D-ID video generation failed: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"latency_ms": time.Since(startTime).Milliseconds(),
		"talk_id":    talkID,
		"report_id":  reportID,
	}).Info("D-ID video generation initiated")

	return talkID, nil
}

// CompleteVideoGeneration polls D-ID for video completion and uploads to S3
func (a *AIService) CompleteVideoGeneration(reportID, jobID string, rssService interface{}) error {
	rs, ok := rssService.(*RSSService)
	if !ok {
		return fmt.Errorf("invalid rss service type")
	}

	// Mock job
	if isMockVideoJob(jobID) {
		logrus.WithField("job_id", jobID).Info("Handling mock video completion")
		time.Sleep(10 * time.Second)
		mockURL := "https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/BigBuckBunny.mp4"
		return rs.UpdateReportVideoStatus(reportID, jobID, "completed", mockURL)
	}

	// HeyGen job — poll until done, then optionally upload to S3
	if a.heygenService != nil {
		logrus.WithField("video_id", jobID).Info("Polling HeyGen for video completion")
		videoURL, err := a.heygenService.WaitForVideo(jobID)
		if err != nil {
			_ = rs.UpdateReportVideoStatus(reportID, jobID, "failed", "")
			return err
		}

		// Upload to S3 if available, otherwise use HeyGen URL directly
		finalURL := videoURL
		if a.s3Service != nil {
			s3Key := fmt.Sprintf("videos/%s/video.mp4", reportID)
			uploaded, err := a.s3Service.DownloadAndUploadVideo(videoURL, s3Key)
			if err != nil {
				logrus.WithError(err).Warn("Failed to upload HeyGen video to S3, using HeyGen URL")
			} else {
				finalURL = uploaded
			}
		}

		logrus.WithFields(logrus.Fields{
			"report_id": reportID,
			"video_url": finalURL,
		}).Info("HeyGen video completed")
		return rs.UpdateReportVideoStatus(reportID, jobID, "completed", finalURL)
	}

	// D-ID fallback
	if a.didService == nil || a.s3Service == nil {
		return fmt.Errorf("required services not configured")
	}

	maxAttempts := 60
	for attempt := 0; attempt < maxAttempts; attempt++ {
		videoStatus, err := a.didService.GetVideoStatus(jobID)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		logrus.WithFields(logrus.Fields{"talk_id": jobID, "status": videoStatus.Status}).Info("D-ID status check")

		if videoStatus.Status == "done" || videoStatus.Status == "completed" {
			resultURL := videoStatus.ResultURL
			if resultURL == "" {
				resultURL = videoStatus.VideoURL
			}
			if resultURL != "" {
				s3Key := fmt.Sprintf("videos/%s/video.mp4", reportID)
				videoURL, err := a.s3Service.DownloadAndUploadVideo(resultURL, s3Key)
				if err != nil {
					videoURL = resultURL
				}
				return rs.UpdateReportVideoStatus(reportID, jobID, "completed", videoURL)
			}
		}
		if videoStatus.Status == "failed" || videoStatus.Status == "error" {
			_ = rs.UpdateReportVideoStatus(reportID, jobID, "failed", "")
			return fmt.Errorf("D-ID video generation failed")
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("video generation timeout")
}

// isMockVideoJob checks if a job ID is a mock video job
func isMockVideoJob(jobID string) bool {
	return strings.HasPrefix(jobID, "mock_job_")
}

// mockVideoGeneration provides fallback mock implementation
func (a *AIService) mockVideoGeneration() (string, error) {
	videoJobID := fmt.Sprintf("mock_job_%d", time.Now().Unix())
	logrus.WithField("job_id", videoJobID).Info("Mock video generation started")
	
	// Mock: Complete video after 10 seconds for testing
	go func() {
		time.Sleep(10 * time.Second)
		logrus.WithField("video_job_id", videoJobID).Info("Mock video generation completed")
	}()
	
	return videoJobID, nil
}
// CloneVoiceAndUpload clones a voice from audio data using ElevenLabs, generates TTS audio
// for the given script, uploads the audio to S3, and returns the public S3 URL.
func (a *AIService) CloneVoiceAndUpload(audioData []byte, audioFilename, voiceName, script, reportID string) (string, error) {
	if a.elevenLabsSvc == nil {
		return "", fmt.Errorf("ElevenLabs not configured - add ELEVENLABS_API_KEY to .env")
	}
	if a.s3Service == nil {
		return "", fmt.Errorf("S3 not configured - required to host audio for D-ID")
	}

	voiceID, err := a.elevenLabsSvc.CloneVoice(voiceName, audioData, audioFilename)
	if err != nil {
		return "", fmt.Errorf("voice cloning failed: %w", err)
	}

	ttsScript := script
	if len(ttsScript) > 900 {
		ttsScript = ttsScript[:900]
		if idx := strings.LastIndexAny(ttsScript, ".!?"); idx > 0 {
			ttsScript = ttsScript[:idx+1]
		}
	}

	mp3Data, err := a.elevenLabsSvc.GenerateSpeech(ttsScript, voiceID)
	if err != nil {
		return "", fmt.Errorf("TTS generation failed: %w", err)
	}

	key := fmt.Sprintf("audio/%s/voice.mp3", reportID)
	audioURL, err := a.s3Service.UploadImage(key, mp3Data, "audio/mpeg")
	if err != nil {
		return "", fmt.Errorf("failed to upload audio to S3: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"report_id": reportID,
		"voice_id":  voiceID,
		"audio_url": audioURL,
	}).Info("Cloned voice audio uploaded to S3")

	return audioURL, nil
}

// MockCompleteVideo simulates video completion for testing
// In production, this will be replaced with actual Veo API webhook/polling
func (a *AIService) MockCompleteVideo(reportID, videoJobID string, rssService interface{}) {
	time.Sleep(10 * time.Second)
	
	// Generate mock video URL
	mockVideoURL := fmt.Sprintf("https://storage.googleapis.com/mock-videos/%s.mp4", videoJobID)
	
	// Update report status
	if rs, ok := rssService.(*RSSService); ok {
		err := rs.UpdateReportVideoStatus(reportID, videoJobID, "completed", mockVideoURL)
		if err != nil {
			logrus.WithError(err).Error("Failed to update mock video status")
		} else {
			logrus.WithFields(logrus.Fields{
				"report_id":    reportID,
				"video_job_id": videoJobID,
				"video_url":    mockVideoURL,
			}).Info("Mock video generation completed")
		}
	}
}


// SearchAndGenerateStory searches the internet for information about a topic and generates a comprehensive story
func (a *AIService) SearchAndGenerateStory(query, authorName string) (map[string]interface{}, error) {
	startTime := time.Now()
	
	correspondentIntro := "I'm your correspondent"
	if authorName != "" {
		correspondentIntro = fmt.Sprintf("I'm %s", authorName)
	}
	
	// Step 1: Search for information about the topic
	searchPrompt := fmt.Sprintf(`You are a news researcher. Search your knowledge base for current information about: "%s"

Provide:
1. Key facts and recent developments
2. Main people/organizations involved
3. Global impact and significance
4. Why this matters now

Format your response as a comprehensive news brief.`, query)

	logrus.WithField("query", query).Info("Starting search for story information")
	searchResults, err := a.geminiClient.GenerateContent(searchPrompt)
	if err != nil {
		logrus.WithError(err).Error("Failed to search for story information")
		return nil, fmt.Errorf("search failed: %w", err)
	}
	
	// Step 2: Generate a comprehensive news report
	reportPrompt := fmt.Sprintf(`You are a professional news correspondent named %s writing a comprehensive news report.

TOPIC: %s

RESEARCH FINDINGS:
%s

Generate a comprehensive, engaging news report script that includes:

1. OPENING: Start with "Good evening, %s, reporting live with breaking news."

2. MAIN STORY: Present the topic and key details

3. BACKGROUND CONTEXT: Provide relevant background information (2-3 sentences)

4. ANALYSIS: Explain why this matters and what the implications are (2-3 sentences)

5. GLOBAL REACTIONS: Describe how the world is reacting to this story (2-3 sentences)

6. IMPLICATIONS: Discuss the broader significance and what might happen next (2-3 sentences)

7. CLOSING: End with "Our team will continue monitoring this story as it develops. We'll bring you updates as soon as new information becomes available. This is %s, and I'll be following this story closely. Stay with us for more updates."

IMPORTANT GUIDELINES:
- Write in a professional news anchor style
- Be factual and balanced
- Keep the tone authoritative but accessible
- Make it engaging and informative
- Length: Aim for 6-8 paragraphs
- Do NOT include any image prompts or image generation instructions

Generate the complete news report script now:`, 
		authorName, query, searchResults, correspondentIntro, authorName)

	logrus.WithField("query", query).Info("Starting report generation")
	script, err := a.geminiClient.GenerateContent(reportPrompt)
	if err != nil {
		logrus.WithError(err).Error("Failed to generate report from search")
		return nil, fmt.Errorf("report generation failed: %w", err)
	}
	
	// Strip any image prompts that might have been included
	script = a.stripImagePrompts(script)
	
	// Strip markdown formatting
	script = a.stripMarkdown(script)
	
	logrus.WithFields(logrus.Fields{
		"latency": time.Since(startTime).Milliseconds(),
		"query":   query,
		"author":  authorName,
		"method":  "gemini_search",
	}).Info("Story generated from search query")
	
	return map[string]interface{}{
		"title":       query,
		"description": searchResults,
		"report":      script,
		"script":      script,
		"reactions":   []map[string]interface{}{},
		"url":         "",
	}, nil
}

// GenerateStoryImagesWithImagen generates images for a story using Google Imagen
func (a *AIService) GenerateStoryImagesWithImagen(title, description string) ([]map[string]interface{}, error) {
	if a.imagenService == nil {
		return nil, fmt.Errorf("Google Imagen service not configured")
	}

	imageBase64s, err := a.imagenService.GenerateStoryImages(title, description, a.geminiClient)
	if err != nil {
		return nil, fmt.Errorf("failed to generate images: %w", err)
	}

	var results []map[string]interface{}
	for i, base64Str := range imageBase64s {
		// Default to data URL
		finalURL := fmt.Sprintf("data:image/png;base64,%s", base64Str)
		
		// Upload to S3 if service is available
		if a.s3Service != nil {
			// Decode base64 to bytes
			data, err := base64.StdEncoding.DecodeString(base64Str)
			if err == nil {
				// Create S3 key
				key := fmt.Sprintf("stories/%s/image-%d.png", strings.ReplaceAll(strings.ToLower(title), " ", "-"), i+1)
				
				// Upload to S3
				s3URL, err := a.s3Service.UploadImage(key, data, "image/png")
				if err == nil {
					finalURL = s3URL
					logrus.WithField("s3_url", s3URL).Info("Image uploaded to S3")
				} else {
					logrus.WithError(err).Warn("Failed to upload image to S3, using data URL")
				}
			}
		}
		
		results = append(results, map[string]interface{}{
			"url":    finalURL,
			"type":   "story",
			"source": "google-imagen",
			"index":  i + 1,
		})
	}

	return results, nil
}

// stripMarkdown removes markdown formatting symbols so scripts read cleanly on air
func stripMarkdown(s string) string {
	re := regexp.MustCompile(`\*{1,2}([^*\n]+)\*{1,2}`)
	s = re.ReplaceAllString(s, "$1")
	re = regexp.MustCompile(`_{1,2}([^_\n]+)_{1,2}`)
	s = re.ReplaceAllString(s, "$1")
	re = regexp.MustCompile(`(?m)^#{1,6}\s+`)
	s = re.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "*", "")
	return strings.TrimSpace(s)
}
