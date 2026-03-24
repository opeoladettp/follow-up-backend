package api

import (
	"io"
	"net/http"
	"net/url"
	"strings"

	"followupmedium-newsroom/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func SetupRSSRoutes(router *gin.RouterGroup, rssService *services.RSSService, aiService *services.AIService) {
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
		rss.POST("/generate-video", generateVideo(aiService, rssService))
		
		// Clone voice and generate audio for video
		rss.POST("/clone-voice", cloneVoice(aiService))

		// Proxy remote images (e.g. Google profile pictures) to avoid CORS/referrer issues
		rss.GET("/proxy-image", proxyImage())
		
		// Get report status (including video generation status)
		rss.GET("/report-status/:report_id", getReportStatus(rssService))

		// Get report by story title (for StoryPanel media lookup)
		rss.GET("/report-by-title", getReportByTitle(rssService))

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

func generateVideo(aiService *services.AIService, rssService *services.RSSService) gin.HandlerFunc {
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

		// D-ID requires a public HTTPS URL ending in .jpg/.jpeg/.png
		// UploadAvatarToS3 handles all cases: base64 data URLs, Google profile pics, etc.
		avatarURL, err := aiService.UploadAvatarToS3(request.AvatarURL, request.ReportID)
		if err != nil {
			logrus.WithError(err).Error("Failed to process avatar for D-ID")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process avatar image"})
			return
		}
		logrus.WithFields(logrus.Fields{
			"report_id":  request.ReportID,
			"avatar_url": avatarURL,
		}).Info("Avatar URL ready for D-ID")

		// Trigger video generation pipeline
		videoJobID, err := aiService.TriggerProductionPipeline(request.Script, avatarURL, request.ReportID, request.VoiceAudioURL)
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

		// Generate images with Google Imagen; fall back to placeholders if not configured
		images, err := aiService.GenerateStoryImagesWithImagen(request.Title, request.Description)
		if err != nil {
			logrus.WithError(err).Warn("Image generation unavailable, using placeholders")
			images = []map[string]interface{}{
				{"url": "https://picsum.photos/seed/" + strings.ReplaceAll(request.Title, " ", "-") + "1/800/450", "type": "story", "source": "placeholder", "index": 1},
				{"url": "https://picsum.photos/seed/" + strings.ReplaceAll(request.Title, " ", "-") + "2/800/450", "type": "story", "source": "placeholder", "index": 2},
				{"url": "https://picsum.photos/seed/" + strings.ReplaceAll(request.Title, " ", "-") + "3/800/450", "type": "story", "source": "placeholder", "index": 3},
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"images":    images,
			"reactions": []interface{}{},
		})
	}
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
		c.DataFromReader(resp.StatusCode, resp.ContentLength, contentType, resp.Body, nil)
	}
}
