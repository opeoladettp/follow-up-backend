package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"followupmedium-newsroom/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func SetupRSSRoutes(router *gin.RouterGroup, rssService *services.RSSService, aiService *services.AIService, authService *services.AuthService) {
	rss := router.Group("/rss")
	{
		// RSS Feed Management (Editor/Admin only - add middleware in production)
		rss.GET("/feeds", getRSSFeeds(rssService))
		rss.POST("/feeds", addRSSFeedNew(rssService))
		rss.PUT("/feeds/:id", updateRSSFeed(rssService))
		rss.DELETE("/feeds/:id", deleteRSSFeed(rssService))
		
		// Legacy feed management (keep for backward compatibility)
		rss.POST("/feeds/legacy", addRSSFeed(rssService))
		rss.DELETE("/feeds/legacy", removeRSSFeed(rssService))
		
		// Get all RSS headlines
		rss.GET("/headlines", getRSSHeadlines(rssService))
		
		// Get headlines from specific source
		rss.GET("/headlines/:source", getRSSHeadlinesBySource(rssService))
		
		// Generate news report from headline
		rss.POST("/generate-report", generateNewsReport(rssService, aiService))
		
		// Search and generate story from user query
		rss.POST("/search-story", searchAndGenerateStory(aiService))
		
		// Generate images and reactions for a story
		rss.POST("/generate-media", generateStoryMedia(aiService))
		
		// Refine news report with AI
		rss.POST("/refine-report", refineNewsReport(aiService))
		
		// Save edited report
		rss.POST("/save-report", saveReport(rssService))
		
		// Generate video from report
		rss.POST("/generate-video", generateVideo(aiService, rssService, authService))
		
		// Clone voice and generate audio for video
		rss.POST("/clone-voice", cloneVoice(aiService))

		// Proxy remote images (e.g. Google profile pictures) to avoid CORS/referrer issues
		rss.GET("/proxy-image", proxyImage())
		
		// Get report status (including video generation status)
		rss.GET("/report-status/:report_id", getReportStatus(rssService))

		// Get report by story title (for StoryPanel media lookup)
		rss.GET("/report-by-title", getReportByTitle(rssService))

		// Bust headlines cache (force refetch from all feeds)
		rss.POST("/refresh-headlines", refreshHeadlines(rssService))

		// Update images on an existing report (called after media generation)
		rss.PATCH("/report/:id/images", updateReportImages(rssService))
	}
}

func getRSSHeadlines(rssService *services.RSSService) gin.HandlerFunc {
	return func(c *gin.Context) {
		headlines, err := rssService.FetchAllHeadlines()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"headlines": headlines,
			"count":     len(headlines),
		})
	}
}

func getRSSHeadlinesBySource(rssService *services.RSSService) gin.HandlerFunc {
	return func(c *gin.Context) {
		source := c.Param("source")
		
		headlines, err := rssService.FetchHeadlinesBySource(source)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"source":    source,
			"headlines": headlines,
			"count":     len(headlines),
		})
	}
}

func generateNewsReport(rssService *services.RSSService, aiService *services.AIService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request struct {
			HeadlineID  string `json:"headline_id" binding:"required"`
			Title       string `json:"title" binding:"required"`
			Description string `json:"description"`
			URL         string `json:"url"`
			AuthorName  string `json:"author_name"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Generate news report using Gemini
		report, err := aiService.GenerateNewsReport(request.Title, request.Description, request.URL, request.AuthorName)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"report":      report,
			"headline_id": request.HeadlineID,
		})
	}
}

func saveReport(rssService *services.RSSService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request struct {
			HeadlineID string                   `json:"headline_id" binding:"required"`
			Title      string                   `json:"title" binding:"required"`
			Script     string                   `json:"script" binding:"required"`
			Author     string                   `json:"author"`
			Images     []map[string]interface{} `json:"images"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		reportID, err := rssService.SaveReportWithMedia(request.HeadlineID, request.Title, request.Script, request.Author, request.Images)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":   "Report saved successfully",
			"report_id": reportID,
		})
	}
}

func generateVideo(aiService *services.AIService, rssService *services.RSSService, authService *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request struct {
			ReportID      string `json:"report_id" binding:"required"`
			Script        string `json:"script" binding:"required"`
			AvatarURL     string `json:"avatar_url" binding:"required"`
			Author        string `json:"author"`
			VoiceAudioURL string `json:"voice_audio_url"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Look up the requesting user's HeyGen avatar/voice IDs
		var userHeygenAvatarID, userHeygenVoiceID string
		if userID, exists := c.Get("userID"); exists {
			if u, err := authService.GetUserByID(userID.(string)); err == nil {
				userHeygenAvatarID = u.HeygenAvatarID
				userHeygenVoiceID = u.HeygenVoiceID
			}
		}

		// For HeyGen, skip S3 avatar upload — HeyGen uses its own stock avatars
		avatarURL, err := aiService.PrepareAvatarURL(request.AvatarURL, request.ReportID)
		if err != nil {
			logrus.WithError(err).Error("Failed to process avatar")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process avatar image"})
			return
		}
		logrus.WithFields(logrus.Fields{
			"report_id":         request.ReportID,
			"avatar_url":        avatarURL,
			"heygen_avatar_set": userHeygenAvatarID != "",
			"heygen_voice_set":  userHeygenVoiceID != "",
		}).Info("Avatar URL ready for video generation")

		// Trigger video generation pipeline
		videoJobID, err := aiService.TriggerProductionPipeline(request.Script, avatarURL, request.ReportID, request.VoiceAudioURL, userHeygenAvatarID, userHeygenVoiceID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Update report with video job ID and status
		err = rssService.UpdateReportVideoStatus(request.ReportID, videoJobID, "processing", "")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update report status"})
			return
		}

		// Start async video completion polling and S3 upload
		go func() {
			err := aiService.CompleteVideoGeneration(request.ReportID, videoJobID, rssService)
			if err != nil {
				logrus.WithError(err).WithField("report_id", request.ReportID).Error("Failed to complete video generation")
				// Mark as failed in DB
				_ = rssService.UpdateReportVideoStatus(request.ReportID, videoJobID, "failed", "")
			}
		}()

		c.JSON(http.StatusOK, gin.H{
			"message":      "Video generation started",
			"video_job_id": videoJobID,
			"report_id":    request.ReportID,
			"status":       "processing",
		})
	}
}

func getRSSFeeds(rssService *services.RSSService) gin.HandlerFunc {
	return func(c *gin.Context) {
		feeds, err := rssService.GetRSSFeeds()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"feeds": feeds, "count": len(feeds)})
	}
}

func addRSSFeed(rssService *services.RSSService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request struct {
			FeedURL  string `json:"feed_url" binding:"required"`
			FeedName string `json:"feed_name" binding:"required"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		_, err := rssService.AddRSSFeed(request.FeedURL, request.FeedName, "")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "RSS feed added successfully"})
	}
}

func removeRSSFeed(rssService *services.RSSService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request struct {
			FeedURL string `json:"feed_url" binding:"required"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Find by URL and delete
		feeds, err := rssService.GetRSSFeeds()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		for _, f := range feeds {
			if f.URL == request.FeedURL {
				if err := rssService.DeleteRSSFeed(f.ID.Hex()); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{"message": "RSS feed removed successfully"})
				return
			}
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "feed not found"})
	}
}

func refineNewsReport(aiService *services.AIService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request struct {
			CurrentScript string `json:"current_script" binding:"required"`
			UserFeedback  string `json:"user_feedback" binding:"required"`
			AuthorName    string `json:"author_name"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		refinedScript, err := aiService.RefineNewsReport(request.CurrentScript, request.UserFeedback, request.AuthorName)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"refined_script": refinedScript,
		})
	}
}

func getReportStatus(rssService *services.RSSService) gin.HandlerFunc {
	return func(c *gin.Context) {
		reportID := c.Param("report_id")

		report, err := rssService.GetReportStatus(reportID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"report_id":    report.ID.Hex(),
			"title":        report.Title,
			"author":       report.Author,
			"status":       report.Status,
			"video_status": report.VideoStatus,
			"video_job_id": report.VideoJobID,
			"video_url":    report.VideoURL,
			"created_at":   report.CreatedAt,
			"updated_at":   report.UpdatedAt,
		})
	}
}

func addRSSFeedNew(rssService *services.RSSService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request struct {
			Name     string `json:"name" binding:"required"`
			URL      string `json:"url" binding:"required"`
			Category string `json:"category"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		feed, err := rssService.AddRSSFeed(request.URL, request.Name, request.Category)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":      "RSS feed added successfully",
			"feed":         feed,
			"resolved_url": feed.URL, // show what X handle was converted to
		})
	}
}

func updateRSSFeed(rssService *services.RSSService) gin.HandlerFunc {
	return func(c *gin.Context) {
		feedID := c.Param("id")

		var request struct {
			Name     string `json:"name"`
			Category string `json:"category"`
			Active   *bool  `json:"active"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		feed, err := rssService.UpdateRSSFeed(feedID, request.Name, request.Category, request.Active)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "RSS feed updated successfully", "feed": feed})
	}
}

func deleteRSSFeed(rssService *services.RSSService) gin.HandlerFunc {
	return func(c *gin.Context) {
		feedID := c.Param("id")

		if err := rssService.DeleteRSSFeed(feedID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "RSS feed deleted successfully"})
	}
}


func generateStoryMedia(aiService *services.AIService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request struct {
			Title       string `json:"title" binding:"required"`
			Description string `json:"description"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Try Google Imagen first
		images, err := aiService.GenerateStoryImagesWithImagen(request.Title, request.Description)
		if err != nil {
			logrus.WithError(err).Warn("Imagen unavailable, falling back to web image search")
			// Pass both title and script so the search can extract better keywords
			images = fetchNewsAPIImages(request.Title, request.Description)
		}

		c.JSON(http.StatusOK, gin.H{
			"images":    images,
			"reactions": []interface{}{},
		})
	}
}

// fetchNewsAPIImages queries NewsAPI and Wikipedia for images relevant to the story.
// It uses both the headline title and the generated script to build a focused search query.
func fetchNewsAPIImages(title, script string) []map[string]interface{} {
	newsAPIKey := os.Getenv("NEWSAPI_KEY")
	var images []map[string]interface{}

	// Build the best possible search query from title + script context
	searchQuery := buildImageSearchQuery(title, script)

	if newsAPIKey != "" {
		query := url.QueryEscape(searchQuery)
		apiURL := fmt.Sprintf("https://newsapi.org/v2/everything?q=%s&pageSize=9&sortBy=relevancy&language=en&apiKey=%s", query, newsAPIKey)

		resp, err := http.Get(apiURL) //nolint:gosec
		if err == nil {
			defer resp.Body.Close()
			var result struct {
				Articles []struct {
					URLToImage string `json:"urlToImage"`
					Title      string `json:"title"`
					URL        string `json:"url"`
				} `json:"articles"`
			}
			if json.NewDecoder(resp.Body).Decode(&result) == nil {
				for i, a := range result.Articles {
					if a.URLToImage != "" && strings.HasPrefix(a.URLToImage, "http") {
						images = append(images, map[string]interface{}{
							"url":         a.URLToImage,
							"description": a.Title,
							"source_url":  a.URL,
							"type":        "news",
							"source":      "newsapi",
							"index":       i + 1,
							"attribution": "Image sourced from NewsAPI — verify copyright before broadcast use",
						})
					}
					if len(images) >= 3 {
						break
					}
				}
			}
		}
	}

	if len(images) > 0 {
		return images
	}

	// Fallback: Google Custom Search (returns Creative Commons / licensed images)
	googleAPIKey := os.Getenv("GOOGLE_API_KEY")
	googleCSEID := os.Getenv("GOOGLE_CSE_ID")
	if googleAPIKey != "" && googleCSEID != "" {
		query := url.QueryEscape(searchQuery)
		// searchType=image, rights=cc_publicdomain|cc_attribute|cc_sharealike for reusable images
		cseURL := fmt.Sprintf(
			"https://www.googleapis.com/customsearch/v1?key=%s&cx=%s&q=%s&searchType=image&num=3&rights=cc_publicdomain|cc_attribute|cc_sharealike",
			googleAPIKey, googleCSEID, query,
		)
		resp, err := http.Get(cseURL) //nolint:gosec
		if err == nil {
			defer resp.Body.Close()
			var result struct {
				Items []struct {
					Title   string `json:"title"`
					Link    string `json:"link"`
					Image   struct {
						ContextLink string `json:"contextLink"`
					} `json:"image"`
				} `json:"items"`
			}
			if json.NewDecoder(resp.Body).Decode(&result) == nil {
				for i, item := range result.Items {
					if item.Link != "" {
						images = append(images, map[string]interface{}{
							"url":         item.Link,
							"description": item.Title,
							"source_url":  item.Image.ContextLink,
							"type":        "news",
							"source":      "google-cse",
							"index":       i + 1,
							"attribution": "Creative Commons image — modify before broadcast use to avoid copyright issues",
						})
					}
				}
			}
		}
	}

	if len(images) > 0 {
		return images
	}

	// Final fallback: Wikimedia Commons (always free to use)
	query := url.QueryEscape(searchQuery)
	wikiURL := fmt.Sprintf("https://en.wikipedia.org/w/api.php?action=query&generator=search&gsrsearch=%s&gsrlimit=5&prop=pageimages&piprop=original&format=json", query)
	resp, err := http.Get(wikiURL) //nolint:gosec
	if err == nil {
		defer resp.Body.Close()
		var result struct {
			Query struct {
				Pages map[string]struct {
					Title    string `json:"title"`
					Original struct {
						Source string `json:"source"`
					} `json:"original"`
				} `json:"pages"`
			} `json:"query"`
		}
		if json.NewDecoder(resp.Body).Decode(&result) == nil {
			i := 1
			for _, page := range result.Query.Pages {
				if page.Original.Source != "" {
					images = append(images, map[string]interface{}{
						"url":         page.Original.Source,
						"description": page.Title,
						"type":        "news",
						"source":      "wikipedia",
						"index":       i,
						"attribution": "Wikimedia Commons — free to use with attribution",
					})
					i++
					if i > 3 {
						break
					}
				}
			}
		}
	}

	return images
}

// buildImageSearchQuery extracts the most meaningful search terms from the story title and script.
// It strips HTML, removes generic filler phrases, and focuses on named entities and key topics.
func buildImageSearchQuery(title, script string) string {
	// Strip HTML from both inputs
	title = stripHTMLTags(title)
	script = stripHTMLTags(script)

	// Remove common broadcast filler from the title
	prefixes := []string{
		"STATEHOUSE PRESS RELEASE", "PRESS RELEASE", "BREAKING:", "BREAKING NEWS:",
		"STATEMENT:", "OFFICIAL:", "UPDATE:", "EXCLUSIVE:",
	}
	q := title
	for _, p := range prefixes {
		if strings.HasPrefix(strings.ToUpper(q), p) {
			q = strings.TrimSpace(q[len(p):])
		}
	}

	// If the script is available, extract the first 2 sentences for richer context
	if script != "" {
		sentences := strings.SplitN(script, ".", 4)
		var context strings.Builder
		for i, s := range sentences {
			if i >= 2 {
				break
			}
			s = strings.TrimSpace(s)
			if len(s) > 20 {
				context.WriteString(s)
				context.WriteString(". ")
			}
		}
		// Combine title keywords with script context, capped at 120 chars
		combined := q + " " + context.String()
		if len(combined) > 120 {
			if idx := strings.LastIndex(combined[:120], " "); idx > 0 {
				combined = combined[:idx]
			} else {
				combined = combined[:120]
			}
		}
		return strings.TrimSpace(combined)
	}

	// Title-only fallback — truncate to 80 chars
	if len(q) > 80 {
		if idx := strings.LastIndex(q[:80], " "); idx > 0 {
			q = q[:idx]
		} else {
			q = q[:80]
		}
	}
	return q
}

// stripHTMLTags is a lightweight HTML stripper for use in search query building.
func stripHTMLTags(s string) string {
	re := regexp.MustCompile(`<[^>]+>`)
	s = re.ReplaceAllString(s, " ")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	re = regexp.MustCompile(`\s{2,}`)
	return strings.TrimSpace(re.ReplaceAllString(s, " "))
}


// searchAndGenerateStory searches the internet and generates a story from user query
func searchAndGenerateStory(aiService *services.AIService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request struct {
			Query      string `json:"query" binding:"required"`
			AuthorName string `json:"author_name"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Generate story from search query
		result, err := aiService.SearchAndGenerateStory(request.Query, request.AuthorName)
		if err != nil {
			// Check if it's a network error
			errMsg := err.Error()
			if strings.Contains(errMsg, "no such host") || strings.Contains(errMsg, "connection refused") || strings.Contains(errMsg, "dial tcp") {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"error": "Network connectivity issue: Cannot reach AI service. Please check your internet connection.",
					"details": errMsg,
				})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to generate story",
				"details": errMsg,
			})
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

// cloneVoice accepts a multipart form with an audio file + report_id + script,
// clones the voice via ElevenLabs, generates TTS audio, uploads to S3, and returns the audio URL.
func cloneVoice(aiService *services.AIService) gin.HandlerFunc {
	return func(c *gin.Context) {
		reportID := c.PostForm("report_id")
		script := c.PostForm("script")
		if reportID == "" || script == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "report_id and script are required"})
			return
		}

		file, header, err := c.Request.FormFile("audio")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "audio file is required"})
			return
		}
		defer file.Close()

		audioData, err := io.ReadAll(file)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read audio file"})
			return
		}

		voiceName := c.PostForm("voice_name")
		if voiceName == "" {
			voiceName = "reporter-" + reportID
		}

		audioURL, err := aiService.CloneVoiceAndUpload(audioData, header.Filename, voiceName, script, reportID)
		if err != nil {
			logrus.WithError(err).Error("Voice cloning failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"audio_url": audioURL,
			"message":   "Voice cloned and audio generated successfully",
		})
	}
}

func refreshHeadlines(rssService *services.RSSService) gin.HandlerFunc {
	return func(c *gin.Context) {
		rssService.InvalidateHeadlinesCache()
		c.JSON(http.StatusOK, gin.H{"message": "Headlines cache cleared"})
	}
}

func getReportByTitle(rssService *services.RSSService) gin.HandlerFunc {
	return func(c *gin.Context) {
		title := c.Query("title")
		if title == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "title query param required"})
			return
		}
		report, err := rssService.GetReportByTitle(title)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, report)
	}
}

func updateReportImages(rssService *services.RSSService) gin.HandlerFunc {
	return func(c *gin.Context) {
		reportID := c.Param("id")
		var request struct {
			Images []map[string]interface{} `json:"images" binding:"required"`
		}
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := rssService.UpdateReportImages(reportID, request.Images); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "images updated"})
	}
}

// proxyImage fetches a remote image and streams it back, bypassing CORS/referrer restrictions.
// Usage: GET /api/v1/rss/proxy-image?url=<encoded_url>
func proxyImage() gin.HandlerFunc {
	return func(c *gin.Context) {
		rawURL := c.Query("url")
		if rawURL == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "url parameter required"})
			return
		}

		// Basic validation - only allow http/https
		parsed, err := url.Parse(rawURL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid URL"})
			return
		}

		resp, err := http.Get(rawURL) //nolint:gosec
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "failed to fetch image"})
			return
		}
		defer resp.Body.Close()

		contentType := resp.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "image/jpeg"
		}

		c.Header("Cache-Control", "public, max-age=86400")

		// If download=1 is set, force browser to download instead of open
		if c.Query("download") == "1" {
			filename := c.Query("filename")
			if filename == "" {
				filename = "story-image.jpg"
			}
			c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
		}

		c.DataFromReader(resp.StatusCode, resp.ContentLength, contentType, resp.Body, nil)
	}
}
