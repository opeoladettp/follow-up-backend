package services

import (
	"fmt"
	"html"
	"time"

	"followupmedium-newsroom/internal/database"
	"followupmedium-newsroom/internal/models"

	"github.com/mmcdole/gofeed"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RSSService struct {
	db        *database.MongoDB
	parser    *gofeed.Parser
	rssFeeds  []string
	feedsMap  map[string]string // URL -> Name mapping
}

type Headline struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	URL         string    `json:"url"`
	Source      string    `json:"source"`
	PublishedAt time.Time `json:"published_at"`
	ImageURL    string    `json:"image_url,omitempty"`
}

func NewRSSService(db *database.MongoDB, rssFeeds []string) *RSSService {
	return &RSSService{
		db:       db,
		parser:   gofeed.NewParser(),
		rssFeeds: rssFeeds,
		feedsMap: make(map[string]string),
	}
}

// AddRSSFeed adds a new RSS feed to the service
func (r *RSSService) AddRSSFeed(feedURL, feedName string) error {
	// Add to feeds list if not already present
	for _, existing := range r.rssFeeds {
		if existing == feedURL {
			return fmt.Errorf("feed already exists")
		}
	}

	// Try to validate the feed (but don't fail if it's temporarily unavailable)
	_, err := r.parser.ParseURL(feedURL)
	if err != nil {
		logrus.WithError(err).WithField("feed_url", feedURL).Warn("Feed validation failed, but adding anyway")
	}

	r.rssFeeds = append(r.rssFeeds, feedURL)
	r.feedsMap[feedURL] = feedName

	logrus.WithFields(logrus.Fields{
		"feed_url":  feedURL,
		"feed_name": feedName,
	}).Info("RSS feed added")

	return nil
}

// GetRSSFeeds returns all configured RSS feeds
func (r *RSSService) GetRSSFeeds() []map[string]interface{} {
	feeds := make([]map[string]interface{}, 0, len(r.rssFeeds))
	for i, url := range r.rssFeeds {
		name := r.feedsMap[url]
		if name == "" {
			name = url
		}
		feeds = append(feeds, map[string]interface{}{
			"id":       fmt.Sprintf("feed-%d", i),
			"url":      url,
			"name":     name,
			"category": "General", // Default category
		})
	}
	return feeds
}

// RemoveRSSFeed removes an RSS feed from the service
func (r *RSSService) RemoveRSSFeed(feedURL string) error {
	for i, url := range r.rssFeeds {
		if url == feedURL {
			r.rssFeeds = append(r.rssFeeds[:i], r.rssFeeds[i+1:]...)
			delete(r.feedsMap, feedURL)
			logrus.WithField("feed_url", feedURL).Info("RSS feed removed")
			return nil
		}
	}
	return fmt.Errorf("feed not found")
}

// FetchAllHeadlines fetches headlines from all configured RSS feeds
func (r *RSSService) FetchAllHeadlines() ([]Headline, error) {
	var allHeadlines []Headline

	for _, feedURL := range r.rssFeeds {
		headlines, err := r.fetchHeadlinesFromFeed(feedURL)
		if err != nil {
			logrus.WithError(err).WithField("feed", feedURL).Error("Failed to fetch headlines")
			continue
		}
		allHeadlines = append(allHeadlines, headlines...)
	}

	logrus.WithField("count", len(allHeadlines)).Info("Fetched RSS headlines")
	return allHeadlines, nil
}

// FetchHeadlinesBySource fetches headlines from a specific source
func (r *RSSService) FetchHeadlinesBySource(source string) ([]Headline, error) {
	// Find the feed URL for this source
	var feedURL string
	for _, url := range r.rssFeeds {
		if contains(url, source) {
			feedURL = url
			break
		}
	}

	if feedURL == "" {
		return nil, fmt.Errorf("source not found: %s", source)
	}

	return r.fetchHeadlinesFromFeed(feedURL)
}

func (r *RSSService) fetchHeadlinesFromFeed(feedURL string) ([]Headline, error) {
	feed, err := r.parser.ParseURL(feedURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse feed: %w", err)
	}

	headlines := make([]Headline, 0, len(feed.Items))
	for _, item := range feed.Items {
		headline := Headline{
			ID:          generateHeadlineID(item),
			Title:       html.UnescapeString(item.Title),
			Description: html.UnescapeString(item.Description),
			URL:         item.Link,
			Source:      html.UnescapeString(feed.Title),
		}

		if item.PublishedParsed != nil {
			headline.PublishedAt = *item.PublishedParsed
		}

		if item.Image != nil {
			headline.ImageURL = item.Image.URL
		}

		headlines = append(headlines, headline)
	}

	return headlines, nil
}

// SaveReport saves a correspondent's edited report and creates a Story entry
func (r *RSSService) SaveReport(headlineID, title, script, author string) (string, error) {
	report := models.NewsReport{
		ID:         primitive.NewObjectID(),
		HeadlineID: headlineID,
		Title:      title,
		Script:     script,
		Author:     author,
		Status:     "draft",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	_, err := r.db.NewsReports().InsertOne(nil, report)
	if err != nil {
		return "", fmt.Errorf("failed to save report: %w", err)
	}

	// Also create a Story entry so it appears in the sidebar
	story := models.Story{
		ID:          primitive.NewObjectID(),
		Title:       title,
		Description: script,
		Category:    "news-report",
		Tags:        []string{"rss", "correspondent"},
		Sources: []models.Source{
			{
				Type: "rss",
				URL:  headlineID,
				Name: author,
			},
		},
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err = r.db.Stories().InsertOne(nil, story)
	if err != nil {
		logrus.WithError(err).Warn("Failed to create Story entry, but report was saved")
		// Don't fail the entire operation if Story creation fails
	}

	logrus.WithFields(logrus.Fields{
		"report_id": report.ID.Hex(),
		"story_id":  story.ID.Hex(),
		"author":    author,
		"title":     title,
	}).Info("News report saved with Story entry")

	return report.ID.Hex(), nil
}

func generateHeadlineID(item *gofeed.Item) string {
	// Generate a unique ID from the item's link or GUID
	if item.GUID != "" {
		return item.GUID
	}
	return item.Link
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr || 
	       len(s) > len(substr) && s[len(s)-len(substr):] == substr ||
	       false
}

// UpdateReportVideoStatus updates the video generation status for a report
func (r *RSSService) UpdateReportVideoStatus(reportID, videoJobID, status, videoURL string) error {
	objID, err := primitive.ObjectIDFromHex(reportID)
	if err != nil {
		return fmt.Errorf("invalid report ID: %w", err)
	}

	update := map[string]interface{}{
		"video_job_id": videoJobID,
		"video_status": status,
		"updated_at":   time.Now(),
	}

	if videoURL != "" {
		update["video_url"] = videoURL
	}

	_, err = r.db.NewsReports().UpdateOne(
		nil,
		map[string]interface{}{"_id": objID},
		map[string]interface{}{"$set": update},
	)

	if err != nil {
		return fmt.Errorf("failed to update report: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"report_id":    reportID,
		"video_job_id": videoJobID,
		"status":       status,
	}).Info("Report video status updated")

	return nil
}

// GetReportStatus retrieves the current status of a report including video generation
func (r *RSSService) GetReportStatus(reportID string) (*models.NewsReport, error) {
	objID, err := primitive.ObjectIDFromHex(reportID)
	if err != nil {
		return nil, fmt.Errorf("invalid report ID: %w", err)
	}

	var report models.NewsReport
	err = r.db.NewsReports().FindOne(nil, map[string]interface{}{"_id": objID}).Decode(&report)
	if err != nil {
		return nil, fmt.Errorf("report not found: %w", err)
	}

	return &report, nil
}
