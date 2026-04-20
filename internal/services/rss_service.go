package services

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"strings"
	"time"

	"followupmedium-newsroom/internal/database"
	"followupmedium-newsroom/internal/models"

	"github.com/mmcdole/gofeed"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	cacheKeyFeeds     = "rss:feeds"
	cacheKeyHeadlines = "rss:headlines"
	cacheTTLFeeds     = 5 * time.Minute
	cacheTTLHeadlines = 10 * time.Minute
)

type RSSService struct {
	db     *database.MongoDB
	redis  *database.Redis
	parser *gofeed.Parser
}

type Headline struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	URL         string    `json:"url"`
	Source      string    `json:"source"`
	Category    string    `json:"category"`
	PublishedAt time.Time `json:"published_at"`
	ImageURL    string    `json:"image_url,omitempty"`
}

func NewRSSService(db *database.MongoDB, redis *database.Redis, rssFeeds []string) *RSSService {
	svc := &RSSService{
		db:     db,
		redis:  redis,
		parser: gofeed.NewParser(),
	}
	svc.cleanupLegacyFeeds()
	return svc
}


func (r *RSSService) col() *mongo.Collection {
	return r.db.Database.Collection("rss_feeds")
}

// cleanupLegacyFeeds removes broken legacy feed entries on startup
func (r *RSSService) cleanupLegacyFeeds() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// Remove legacy string-ID feeds
	_, _ = r.col().DeleteMany(ctx, bson.M{"_id": bson.M{"$type": "string"}})
	// Remove broken public RSSHub Twitter feeds
	_, _ = r.col().DeleteMany(ctx, bson.M{"url": bson.M{"$regex": "rsshub\\.app/twitter"}})
	_, _ = r.col().DeleteMany(ctx, bson.M{"url": bson.M{"$regex": "nitter\\."}})
}

// GetRSSFeeds returns all feeds, using Redis cache
func (r *RSSService) GetRSSFeeds() ([]models.RSSFeed, error) {
	// Try cache first
	if cached, err := r.redis.GetCachedJSON(cacheKeyFeeds); err == nil {
		var feeds []models.RSSFeed
		if json.Unmarshal([]byte(cached), &feeds) == nil {
			return feeds, nil
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := r.col().Find(ctx, bson.M{}, options.Find().SetSort(bson.M{"created_at": 1}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var feeds []models.RSSFeed
	if err = cursor.All(ctx, &feeds); err != nil {
		return nil, err
	}

	// Cache result
	if data, err := json.Marshal(feeds); err == nil {
		_ = r.redis.CacheJSON(cacheKeyFeeds, string(data), cacheTTLFeeds)
	}

	return feeds, nil
}

// AddRSSFeed persists a new feed and invalidates cache
func (r *RSSService) AddRSSFeed(feedURL, feedName, category string) (*models.RSSFeed, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Convert X.com/Twitter handles to Nitter RSS
	var err error
	feedURL, err = normalizeToRSSURL(feedURL)
	if err != nil {
		return nil, err
	}

	var existing models.RSSFeed
	if err := r.col().FindOne(ctx, bson.M{"url": feedURL}).Decode(&existing); err == nil {
		return nil, fmt.Errorf("this feed already exists (added as: %s)", feedURL)
	}

	if category == "" {
		category = "General"
	}

	feed := models.RSSFeed{
		ID:        primitive.NewObjectID(),
		Name:      feedName,
		URL:       feedURL,
		Category:  category,
		Active:    true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if _, err := r.col().InsertOne(ctx, feed); err != nil {
		return nil, err
	}

	r.invalidateCaches()
	logrus.WithFields(logrus.Fields{"url": feedURL, "name": feedName}).Info("RSS feed added")
	return &feed, nil
}

// normalizeToRSSURL validates and normalises a feed URL.
// Twitter/X handles are routed through the self-hosted RSSHub instance (RSSHUB_URL env var).
func normalizeToRSSURL(input string) (string, error) {
	input = strings.TrimSpace(input)

	rsshubBase := os.Getenv("RSSHUB_URL")
	if rsshubBase == "" {
		rsshubBase = "https://rsshub.app"
	}
	rsshubBase = strings.TrimRight(rsshubBase, "/")

	// @handle or bare word without dots — treat as Twitter handle
	if strings.HasPrefix(input, "@") {
		handle := strings.TrimPrefix(input, "@")
		if handle != "" {
			return rsshubBase + "/twitter/user/" + handle, nil
		}
	}

	// twitter.com or x.com profile URL
	if strings.Contains(input, "twitter.com/") || strings.Contains(input, "x.com/") {
		parts := strings.Split(strings.TrimRight(input, "/"), "/")
		handle := parts[len(parts)-1]
		if handle != "" {
			return rsshubBase + "/twitter/user/" + handle, nil
		}
	}

	// Already a proper URL
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		return input, nil
	}

	// Bare domain without scheme
	return "https://" + input, nil
}

// UpdateRSSFeed updates a feed by ID and invalidates cache
func (r *RSSService) UpdateRSSFeed(feedID, name, category string, active *bool) (*models.RSSFeed, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objID, err := primitive.ObjectIDFromHex(feedID)
	if err != nil {
		return nil, fmt.Errorf("invalid feed ID")
	}

	update := bson.M{"updated_at": time.Now()}
	if name != "" {
		update["name"] = name
	}
	if category != "" {
		update["category"] = category
	}
	if active != nil {
		update["active"] = *active
	}

	after := options.After
	var updated models.RSSFeed
	err = r.col().FindOneAndUpdate(
		ctx,
		bson.M{"_id": objID},
		bson.M{"$set": update},
		options.FindOneAndUpdate().SetReturnDocument(after),
	).Decode(&updated)
	if err != nil {
		return nil, fmt.Errorf("feed not found")
	}

	r.invalidateCaches()
	return &updated, nil
}

// DeleteRSSFeed removes a feed and invalidates cache
func (r *RSSService) DeleteRSSFeed(feedID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objID, err := primitive.ObjectIDFromHex(feedID)
	if err != nil {
		return fmt.Errorf("invalid feed ID")
	}

	result, err := r.col().DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return fmt.Errorf("feed not found")
	}

	r.invalidateCaches()
	return nil
}

func (r *RSSService) invalidateCaches() {
	_ = r.redis.InvalidateCache(cacheKeyFeeds)
	_ = r.redis.InvalidateCache(cacheKeyHeadlines)
}

func (r *RSSService) InvalidateHeadlinesCache() {
	_ = r.redis.InvalidateCache(cacheKeyHeadlines)
}
func (r *RSSService) FetchAllHeadlines() ([]Headline, error) {
	// Try headlines cache
	if cached, err := r.redis.GetCachedJSON(cacheKeyHeadlines); err == nil {
		var headlines []Headline
		if json.Unmarshal([]byte(cached), &headlines) == nil {
			logrus.Info("Serving headlines from cache")
			return headlines, nil
		}
	}

	feeds, err := r.GetRSSFeeds()
	if err != nil {
		return nil, fmt.Errorf("failed to load feeds from database: %w", err)
	}
	if len(feeds) == 0 {
		return []Headline{}, nil
	}

	urls := make([]string, 0, len(feeds))
	for _, f := range feeds {
		if f.Active {
			urls = append(urls, f.URL)
		}
	}

	headlines, err := r.fetchFromURLs(urls, nil)
	if err != nil {
		return nil, err
	}

	// Attach category from DB feed
	for i, h := range headlines {
		for _, f := range feeds {
			if strings.Contains(h.Source, f.Name) || strings.Contains(f.URL, strings.ToLower(h.Source)) {
				headlines[i].Category = f.Category
				break
			}
		}
	}

	// Cache headlines
	if data, err := json.Marshal(headlines); err == nil {
		_ = r.redis.CacheJSON(cacheKeyHeadlines, string(data), cacheTTLHeadlines)
	}

	return headlines, nil
}

func (r *RSSService) fetchFromURLs(urls []string, _ map[string]string) ([]Headline, error) {
	rsshubBase := os.Getenv("RSSHUB_URL")
	if rsshubBase == "" {
		rsshubBase = "https://rsshub.app"
	}
	rsshubBase = strings.TrimRight(rsshubBase, "/")

	var all []Headline
	for _, feedURL := range urls {
		// Convert legacy twitter:// scheme to RSSHub URL
		if strings.HasPrefix(feedURL, "twitter://") {
			handle := strings.TrimPrefix(feedURL, "twitter://")
			feedURL = rsshubBase + "/twitter/user/" + handle
		}
		items, err := r.fetchHeadlinesFromFeed(feedURL, "")
		if err != nil {
			logrus.WithError(err).WithField("feed", feedURL).Warn("Failed to fetch headlines from feed, skipping")
			continue
		}
		all = append(all, items...)
	}
	logrus.WithField("count", len(all)).Info("Fetched RSS headlines")
	return all, nil
}

// FetchHeadlinesBySource fetches from a specific source
func (r *RSSService) FetchHeadlinesBySource(source string) ([]Headline, error) {
	feeds, _ := r.GetRSSFeeds()
	for _, f := range feeds {
		if strings.Contains(strings.ToLower(f.URL), strings.ToLower(source)) ||
			strings.Contains(strings.ToLower(f.Name), strings.ToLower(source)) {
			return r.fetchHeadlinesFromFeed(f.URL, f.Category)
		}
	}
	for _, url := range r.seedFeeds {
		if strings.Contains(url, source) {
			return r.fetchHeadlinesFromFeed(url, "")
		}
	}
	return nil, fmt.Errorf("source not found: %s", source)
}

func (r *RSSService) fetchHeadlinesFromFeed(feedURL, category string) ([]Headline, error) {
	feed, err := r.parser.ParseURL(feedURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse feed: %w", err)
	}

	headlines := make([]Headline, 0, len(feed.Items))
	for _, item := range feed.Items {
		h := Headline{
			ID:          generateHeadlineID(item),
			Title:       html.UnescapeString(item.Title),
			Description: html.UnescapeString(item.Description),
			URL:         item.Link,
			Source:      html.UnescapeString(feed.Title),
			Category:    category,
		}
		if item.PublishedParsed != nil {
			h.PublishedAt = *item.PublishedParsed
		}
		if item.Image != nil {
			h.ImageURL = item.Image.URL
		}
		headlines = append(headlines, h)
	}
	return headlines, nil
}

// SaveReport saves a report and creates a Story entry
func (r *RSSService) SaveReport(headlineID, title, script, author string) (string, error) {
	return r.SaveReportWithMedia(headlineID, title, script, author, nil)
}

// SaveReportWithMedia saves a report with optional images and creates a Story entry
func (r *RSSService) SaveReportWithMedia(headlineID, title, script, author string, images []map[string]interface{}) (string, error) {
	report := models.NewsReport{
		ID:         primitive.NewObjectID(),
		HeadlineID: headlineID,
		Title:      title,
		Script:     script,
		Author:     author,
		Images:     images,
		Status:     "draft",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if _, err := r.db.NewsReports().InsertOne(nil, report); err != nil {
		return "", fmt.Errorf("failed to save report: %w", err)
	}

	story := models.Story{
		ID:          primitive.NewObjectID(),
		Title:       title,
		Description: script,
		Category:    "news-report",
		Tags:        []string{"rss", "correspondent"},
		Sources:     []models.Source{{Type: "rss", URL: headlineID, Name: author}},
		Status:      "active",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if _, err := r.db.Stories().InsertOne(nil, story); err != nil {
		logrus.WithError(err).Warn("Failed to create Story entry")
	}

	return report.ID.Hex(), nil
}

// GetReportByTitle finds the most recent NewsReport matching a title
func (r *RSSService) GetReportByTitle(title string) (*models.NewsReport, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := options.FindOne().SetSort(bson.M{"created_at": -1})
	var report models.NewsReport
	err := r.db.NewsReports().FindOne(ctx, bson.M{"title": title}, opts).Decode(&report)
	if err != nil {
		return nil, fmt.Errorf("report not found: %w", err)
	}
	return &report, nil
}

// UpdateReportImages patches the images array on an existing report
func (r *RSSService) UpdateReportImages(reportID string, images []map[string]interface{}) error {
	objID, err := primitive.ObjectIDFromHex(reportID)
	if err != nil {
		return fmt.Errorf("invalid report ID: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = r.db.NewsReports().UpdateOne(ctx,
		bson.M{"_id": objID},
		bson.M{"$set": bson.M{"images": images, "updated_at": time.Now()}},
	)
	return err
}

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
	_, err = r.db.NewsReports().UpdateOne(nil,
		map[string]interface{}{"_id": objID},
		map[string]interface{}{"$set": update},
	)
	return err
}

func (r *RSSService) GetReportStatus(reportID string) (*models.NewsReport, error) {
	objID, err := primitive.ObjectIDFromHex(reportID)
	if err != nil {
		return nil, fmt.Errorf("invalid report ID: %w", err)
	}
	var report models.NewsReport
	if err = r.db.NewsReports().FindOne(nil, map[string]interface{}{"_id": objID}).Decode(&report); err != nil {
		return nil, fmt.Errorf("report not found: %w", err)
	}
	return &report, nil
}

func generateHeadlineID(item *gofeed.Item) string {
	if item.GUID != "" {
		return item.GUID
	}
	return item.Link
}
