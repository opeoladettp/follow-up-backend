package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"followupmedium-newsroom/internal/models"

	"github.com/mmcdole/gofeed"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

type ContentFetcher struct {
	rssParser   *gofeed.Parser
	rateLimiter *rate.Limiter
	httpClient  *http.Client
}

func NewContentFetcher() *ContentFetcher {
	return &ContentFetcher{
		rssParser:   gofeed.NewParser(),
		rateLimiter: rate.NewLimiter(rate.Every(time.Second), 10), // 10 requests per second
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (cf *ContentFetcher) FetchContent(source models.Source) (string, error) {
	// Rate limiting
	if err := cf.rateLimiter.Wait(context.Background()); err != nil {
		return "", fmt.Errorf("rate limiter error: %w", err)
	}

	switch source.Type {
	case "rss":
		return cf.fetchRSSContent(source)
	case "nitter":
		return cf.fetchNitterContent(source)
	case "reddit":
		return cf.fetchRedditContent(source)
	case "twitter":
		// Deprecated: Use Nitter instead
		logrus.Warn("Twitter API is deprecated. Use Nitter instead for free Twitter content.")
		return "", fmt.Errorf("twitter API not configured - use nitter instead")
	default:
		return "", fmt.Errorf("unsupported source type: %s", source.Type)
	}
}

func (cf *ContentFetcher) fetchRSSContent(source models.Source) (string, error) {
	logrus.WithField("url", source.URL).Debug("Fetching RSS content")
	
	feed, err := cf.rssParser.ParseURL(source.URL)
	if err != nil {
		return "", fmt.Errorf("failed to parse RSS feed: %w", err)
	}

	if len(feed.Items) == 0 {
		return "", nil
	}

	// Get the latest item
	latestItem := feed.Items[0]
	
	// Combine title and description for content
	content := strings.TrimSpace(latestItem.Title)
	if latestItem.Description != "" {
		content += "\n\n" + strings.TrimSpace(latestItem.Description)
	}

	logrus.WithFields(logrus.Fields{
		"source": source.Name,
		"title":  latestItem.Title,
		"length": len(content),
	}).Debug("RSS content fetched")

	return content, nil
}

func (cf *ContentFetcher) fetchNitterContent(source models.Source) (string, error) {
	// Nitter provides Twitter content via RSS (FREE!)
	// URL format: https://nitter.net/USERNAME/rss
	logrus.WithField("url", source.URL).Debug("Fetching Nitter content")
	
	feed, err := cf.rssParser.ParseURL(source.URL)
	if err != nil {
		return "", fmt.Errorf("failed to parse Nitter RSS feed: %w", err)
	}

	if len(feed.Items) == 0 {
		return "", nil
	}

	// Get the latest tweet
	latestTweet := feed.Items[0]
	
	content := strings.TrimSpace(latestTweet.Title)
	if latestTweet.Description != "" {
		content += "\n\n" + strings.TrimSpace(latestTweet.Description)
	}

	logrus.WithFields(logrus.Fields{
		"source": source.Name,
		"title":  latestTweet.Title,
		"length": len(content),
	}).Debug("Nitter content fetched")

	return content, nil
}

func (cf *ContentFetcher) fetchRedditContent(source models.Source) (string, error) {
	// Reddit JSON API (FREE, no auth needed!)
	// URL format: https://www.reddit.com/r/SUBREDDIT.json
	logrus.WithField("url", source.URL).Debug("Fetching Reddit content")
	
	req, err := http.NewRequest("GET", source.URL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	
	// Reddit requires a User-Agent
	req.Header.Set("User-Agent", "FollowUpMedium/1.0")
	
	resp, err := cf.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch Reddit content: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("reddit API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse Reddit JSON
	var redditResp struct {
		Data struct {
			Children []struct {
				Data struct {
					Title     string `json:"title"`
					Selftext  string `json:"selftext"`
					URL       string `json:"url"`
					Author    string `json:"author"`
					Subreddit string `json:"subreddit"`
				} `json:"data"`
			} `json:"children"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &redditResp); err != nil {
		return "", fmt.Errorf("failed to parse Reddit JSON: %w", err)
	}

	if len(redditResp.Data.Children) == 0 {
		return "", nil
	}

	// Get the top post
	topPost := redditResp.Data.Children[0].Data
	
	content := fmt.Sprintf("[r/%s] %s", topPost.Subreddit, topPost.Title)
	if topPost.Selftext != "" {
		content += "\n\n" + strings.TrimSpace(topPost.Selftext)
	}

	logrus.WithFields(logrus.Fields{
		"source":    source.Name,
		"subreddit": topPost.Subreddit,
		"title":     topPost.Title,
		"length":    len(content),
	}).Debug("Reddit content fetched")

	return content, nil
}

func (cf *ContentFetcher) fetchTwitterContent(source models.Source) (string, error) {
	// Deprecated: Twitter API is expensive ($100/month)
	// Use Nitter instead for free Twitter content
	logrus.Warn("Twitter API is deprecated. Use Nitter for free Twitter content.")
	return "", fmt.Errorf("twitter API not configured - use nitter instead (free)")
}

// FetchMultipleSources fetches content from multiple sources concurrently
func (cf *ContentFetcher) FetchMultipleSources(sources []models.Source) map[string]string {
	results := make(map[string]string)
	resultsChan := make(chan struct {
		source  models.Source
		content string
		err     error
	}, len(sources))

	// Launch goroutines for each source
	for _, source := range sources {
		go func(s models.Source) {
			content, err := cf.FetchContent(s)
			resultsChan <- struct {
				source  models.Source
				content string
				err     error
			}{s, content, err}
		}(source)
	}

	// Collect results
	for i := 0; i < len(sources); i++ {
		result := <-resultsChan
		if result.err != nil {
			logrus.WithFields(logrus.Fields{
				"source": result.source.Name,
				"error":  result.err.Error(),
			}).Error("Failed to fetch content from source")
			continue
		}
		
		if result.content != "" {
			results[result.source.Name] = result.content
		}
	}

	return results
}