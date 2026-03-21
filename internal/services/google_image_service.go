package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

type GoogleImageService struct {
	geminiClient *GeminiClient
}

type GeneratedImage struct {
	URL         string `json:"url"`
	Prompt      string `json:"prompt"`
	Type        string `json:"type"` // "story", "reaction", "background"
	Source      string `json:"source"`
	Description string `json:"description"`
}

type SocialReaction struct {
	Platform    string `json:"platform"` // twitter, facebook, instagram, reddit
	Author      string `json:"author"`
	Content     string `json:"content"`
	Likes       int    `json:"likes"`
	Shares      int    `json:"shares"`
	ImageURL    string `json:"image_url"`
	Timestamp   string `json:"timestamp"`
}

func NewGoogleImageService(geminiClient *GeminiClient) *GoogleImageService {
	return &GoogleImageService{
		geminiClient: geminiClient,
	}
}

// GenerateStoryImage generates contextual image descriptions using Gemini
// Now includes entity extraction to make images story-specific
func (g *GoogleImageService) GenerateStoryImage(title, description string) (*GeneratedImage, error) {
	// First, extract key entities (people, companies, locations) from the story
	entityPrompt := fmt.Sprintf(`Extract key entities from this news story. Focus on:
1. Notable people mentioned (names, titles, roles)
2. Companies or organizations
3. Locations or places
4. Key objects or concepts

HEADLINE: %s
DESCRIPTION: %s

Return a comma-separated list of the most important entities. Example: "Mark Zuckerberg, Facebook, courtroom, grieving parents"
Return ONLY the list, nothing else.`, title, description)

	entities, err := g.geminiClient.GenerateContent(entityPrompt)
	if err != nil {
		logrus.WithError(err).Warn("Failed to extract entities")
		entities = ""
	}

	// Use Gemini to understand the story and generate a contextual image prompt
	prompt := fmt.Sprintf(`Analyze this news story and generate a detailed, specific image prompt for professional news illustration:

HEADLINE: %s
DESCRIPTION: %s
KEY ENTITIES: %s

Generate an image prompt that:
1. Is SPECIFIC to this story (not generic)
2. Shows key elements and people mentioned in the story
3. Is suitable for professional news media
4. Includes: composition, style, mood, lighting, key visual elements
5. References specific people, places, or objects from the story when relevant
6. Is under 150 words

Examples of GOOD prompts:
- For a tech story with Mark Zuckerberg: "Mark Zuckerberg in a modern tech office, professional lighting, serious expression, corporate setting"
- For a court story: "Courtroom scene with lawyers and judges, formal atmosphere, dramatic lighting, wooden furniture"
- For a business story: "Corporate executives in a boardroom meeting, professional attire, modern office, serious discussion"

Return ONLY the image prompt, nothing else.`, title, description, entities)

	imagePrompt, err := g.geminiClient.GenerateContent(prompt)
	if err != nil {
		logrus.WithError(err).Error("Failed to generate story image prompt")
		return nil, err
	}

	// Also search for relevant images related to the story
	imageSearchPrompt := fmt.Sprintf(`Find relevant images or visual references for this news story:

HEADLINE: %s
DESCRIPTION: %s
KEY ENTITIES: %s

Suggest:
1. Key visual elements to search for (e.g., specific people, locations, objects mentioned)
2. Stock photo keywords (e.g., "tech executive", "courtroom", "emotional moment")
3. Relevant visual themes and settings

Format as simple list.`, title, description, entities)

	imageSearchTerms, err := g.geminiClient.GenerateContent(imageSearchPrompt)
	if err != nil {
		logrus.WithError(err).Warn("Failed to generate image search terms")
		imageSearchTerms = ""
	}

	return &GeneratedImage{
		URL:         fmt.Sprintf("https://via.placeholder.com/1024x768?text=%s", strings.ReplaceAll(title[:40], " ", "+")),
		Prompt:      imagePrompt,
		Type:        "story",
		Source:      "google-generative-ai",
		Description: fmt.Sprintf("Professional news illustration for: %s\n\nSearch terms: %s", title, imageSearchTerms),
	}, nil
}

// GenerateReactionImage creates a visual description for social media reactions
func (g *GoogleImageService) GenerateReactionImage(reactions []SocialReaction) (*GeneratedImage, error) {
	reactionSummary := fmt.Sprintf("Create a detailed image prompt for a social media reactions summary showing %d reactions from various platforms. Include reaction counts, sentiment indicators, and platform logos.", len(reactions))

	imagePrompt, err := g.geminiClient.GenerateContent(reactionSummary)
	if err != nil {
		logrus.WithError(err).Error("Failed to generate reaction image prompt")
		return nil, err
	}

	return &GeneratedImage{
		URL:         "https://via.placeholder.com/1024x768?text=Social+Reactions",
		Prompt:      imagePrompt,
		Type:        "reaction",
		Source:      "google-generative-ai",
		Description: "Social media reactions summary",
	}, nil
}

// GenerateBackgroundImage generates a contextual background image description
func (g *GoogleImageService) GenerateBackgroundImage(context string) (*GeneratedImage, error) {
	prompt := fmt.Sprintf(`Create a detailed image prompt for a professional background image for news context: "%s"
	
The image should be subtle, modern, and suitable as an article background.
Include: style, composition, mood, and color palette.
Keep it under 100 words.
Return ONLY the image prompt, nothing else.`, context)

	imagePrompt, err := g.geminiClient.GenerateContent(prompt)
	if err != nil {
		logrus.WithError(err).Error("Failed to generate background image prompt")
		return nil, err
	}

	return &GeneratedImage{
		URL:         "https://via.placeholder.com/1024x768?text=Background",
		Prompt:      imagePrompt,
		Type:        "background",
		Source:      "google-generative-ai",
		Description: "Background context image",
	}, nil
}

// GenerateSocialReactions uses Gemini to generate realistic social media reactions for a story
func (g *GoogleImageService) GenerateSocialReactions(title, description string) []SocialReaction {
	// Use Gemini to generate realistic social media reactions
	searchPrompt := fmt.Sprintf(`Generate realistic social media reactions and expert commentary about this news story. Create reactions that sound like real people commenting on social media.

HEADLINE: %s
DESCRIPTION: %s

For each reaction, provide:
- Platform (twitter, reddit, linkedin, facebook)
- Author name or handle (realistic person/account name)
- Actual comment/reaction text (what a real person would say about this story - make it specific to the story)
- Engagement metrics (likes, shares, retweets) - realistic numbers
- Time posted (relative time like "2 hours ago", "30 minutes ago")

Create 5-7 reactions total from different platforms. Make them sound authentic and varied:
- Some should be expert opinions
- Some should be casual reactions
- Some should be critical perspectives
- Some should be supportive comments
- Reference specific details from the story in the comments

Format as JSON array with fields: platform, author, content, likes, shares, timestamp

Example format:
[
  {"platform": "twitter", "author": "TechJournalist", "content": "This development could reshape the entire industry. The implications are massive.", "likes": 1240, "shares": 340, "timestamp": "2 hours ago"},
  {"platform": "reddit", "author": "u/NewsReader", "content": "Finally! People have been waiting for this announcement for years.", "likes": 2100, "shares": 450, "timestamp": "1 hour ago"}
]

Return ONLY valid JSON, no other text.`, title, description)

	reactionsJSON, err := g.geminiClient.GenerateContent(searchPrompt)
	if err != nil {
		logrus.WithError(err).Error("Failed to generate social reactions with Gemini")
		// Fallback to generic reactions
		return g.generateFallbackReactions(title)
	}

	// Parse the JSON response
	reactions := parseReactionsJSON(reactionsJSON)
	if len(reactions) > 0 {
		return reactions
	}

	// If parsing failed, return fallback
	return g.generateFallbackReactions(title)
}

// generateFallbackReactions provides fallback reactions if Gemini search fails
func (g *GoogleImageService) generateFallbackReactions(title string) []SocialReaction {
	return []SocialReaction{
		{
			Platform:  "twitter",
			Author:    "NewsWatcher",
			Content:   fmt.Sprintf("Breaking: %s - This is significant!", title),
			Likes:     1240,
			Shares:    340,
			Timestamp: "2 hours ago",
		},
		{
			Platform:  "reddit",
			Author:    "u/NewsReader",
			Content:   fmt.Sprintf("Discussion: %s - Share your thoughts", title),
			Likes:     2100,
			Shares:    450,
			Timestamp: "1 hour ago",
		},
		{
			Platform:  "facebook",
			Author:    "News Network",
			Content:   fmt.Sprintf("📰 %s - Read the full story", title),
			Likes:     3400,
			Shares:    890,
			Timestamp: "30 minutes ago",
		},
	}
}

// parseReactionsJSON parses JSON reactions from Gemini response
func parseReactionsJSON(jsonStr string) []SocialReaction {
	var reactions []SocialReaction
	
	// Try to unmarshal the JSON
	err := json.Unmarshal([]byte(jsonStr), &reactions)
	if err != nil {
		logrus.WithError(err).Warn("Failed to parse reactions JSON")
		return []SocialReaction{}
	}
	
	return reactions
}

// FormatReactionsAsHTML creates HTML representation of reactions for embedding in articles
func (g *GoogleImageService) FormatReactionsAsHTML(reactions []SocialReaction) string {
	html := `<div class="social-reactions-container">
  <h3>Social Media Reactions</h3>
  <div class="reactions-grid">`

	for _, reaction := range reactions {
		html += fmt.Sprintf(`
  <div class="reaction-card platform-%s">
    <div class="platform-badge">%s</div>
    <div class="author">@%s</div>
    <div class="content">%s</div>
    <div class="stats">
      <span class="likes">❤️ %d</span>
      <span class="shares">↗️ %d</span>
    </div>
    <div class="timestamp">%s</div>
  </div>`, reaction.Platform, strings.ToUpper(reaction.Platform), reaction.Author, reaction.Content, reaction.Likes, reaction.Shares, reaction.Timestamp)
	}

	html += `
  </div>
</div>`

	return html
}

// GenerateImagePromptWithGemini uses Gemini to create detailed image prompts
// These prompts can be used with Google Imagen or other image generation APIs
func (g *GoogleImageService) GenerateImagePromptWithGemini(ctx context.Context, topic, style string) (string, error) {
	prompt := fmt.Sprintf(`Create a detailed, professional image prompt for: %s
Style: %s

The prompt should be:
- Specific and descriptive
- Suitable for professional image generation
- Include composition, lighting, mood, and key elements
- Under 150 words

Return ONLY the image prompt.`, topic, style)

	imagePrompt, err := g.geminiClient.GenerateContent(prompt)
	if err != nil {
		return "", err
	}

	return imagePrompt, nil
}

// GetImagePrompts returns contextual image prompts for different story elements
func (g *GoogleImageService) GetImagePrompts(title, description string) map[string]string {
	prompts := make(map[string]string)

	// Extract entities first
	entityPrompt := fmt.Sprintf(`Extract key entities from this news story:
HEADLINE: %s
DESCRIPTION: %s
Return a comma-separated list. Return ONLY the list.`, title, description)

	entities, err := g.geminiClient.GenerateContent(entityPrompt)
	if err != nil {
		entities = ""
	}

	// Use Gemini to generate contextual prompts
	storyPromptRequest := fmt.Sprintf(`Create a specific, contextual image prompt for this news story (NOT generic):

HEADLINE: %s
DESCRIPTION: %s
KEY ENTITIES: %s

The prompt should:
- Reference specific elements from the story
- Be suitable for professional news media
- Include composition, lighting, mood, style
- Be under 150 words

Return ONLY the prompt.`, title, description, entities)

	storyPrompt, err := g.geminiClient.GenerateContent(storyPromptRequest)
	if err != nil {
		storyPrompt = fmt.Sprintf("Professional news illustration for: %s", title)
	}
	prompts["story"] = storyPrompt

	// Reaction image prompt
	reactionPromptRequest := fmt.Sprintf(`Create an image prompt for visualizing social media reactions to this story:

HEADLINE: %s
KEY ENTITIES: %s

The image should show:
- Multiple social media platform indicators
- Engagement metrics and sentiment
- Professional, modern design
- Relevant to the story topic and entities

Return ONLY the prompt.`, title, entities)

	reactionPrompt, err := g.geminiClient.GenerateContent(reactionPromptRequest)
	if err != nil {
		reactionPrompt = "Social media reactions visualization with platform logos, engagement metrics, and sentiment indicators"
	}
	prompts["reaction"] = reactionPrompt

	// Background prompt
	backgroundPrompt := `Subtle professional background for news article
Abstract, modern design
Soft colors and gradients
Suitable as article background without overwhelming content
Style: Minimalist, professional
Mood: Calm, trustworthy`
	prompts["background"] = backgroundPrompt

	return prompts
}
