package services

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"followupmedium-newsroom/internal/database"
	"followupmedium-newsroom/internal/models"

	"github.com/sirupsen/logrus"
)

type DiffEngine struct {
	redis *database.Redis
}

func NewDiffEngine(redis *database.Redis) *DiffEngine {
	return &DiffEngine{
		redis: redis,
	}
}

type DiffResult struct {
	IsNew       bool
	ContentHash string
	DiffType    string // "breaking", "follow_up", "duplicate"
}

// ComputeContentDiff analyzes new content against existing story baseline
func (d *DiffEngine) ComputeContentDiff(storyID string, content string, source models.Source) (*DiffResult, error) {
	startTime := time.Now()
	defer func() {
		logrus.WithFields(logrus.Fields{
			"story_id": storyID,
			"latency":  time.Since(startTime).Milliseconds(),
		}).Debug("Diff computation completed")
	}()

	// Generate content hash
	contentHash := d.generateContentHash(content)
	
	// Check if this exact content was already processed
	isProcessed, err := d.redis.IsProcessed(storyID, contentHash)
	if err != nil {
		return nil, fmt.Errorf("failed to check processed status: %w", err)
	}

	if isProcessed {
		return &DiffResult{
			IsNew:       false,
			ContentHash: contentHash,
			DiffType:    "duplicate",
		}, nil
	}

	// Get previous content hash for comparison
	previousHash, err := d.redis.GetStoryHash(storyID)
	if err != nil && err.Error() != "redis: nil" {
		return nil, fmt.Errorf("failed to get previous hash: %w", err)
	}

	diffType := d.determineDiffType(content, previousHash == "")
	
	// Mark as processed and update hash
	if err := d.redis.AddProcessedID(storyID, contentHash); err != nil {
		logrus.WithError(err).Warn("Failed to mark content as processed")
	}

	if err := d.redis.SetStoryHash(storyID, contentHash); err != nil {
		logrus.WithError(err).Warn("Failed to update story hash")
	}

	return &DiffResult{
		IsNew:       true,
		ContentHash: contentHash,
		DiffType:    diffType,
	}, nil
}

func (d *DiffEngine) generateContentHash(content string) string {
	// Normalize content for consistent hashing
	normalized := strings.ToLower(strings.TrimSpace(content))
	hash := sha256.Sum256([]byte(normalized))
	return fmt.Sprintf("%x", hash)
}

func (d *DiffEngine) determineDiffType(content string, isFirstContent bool) string {
	if isFirstContent {
		return "breaking"
	}

	// Simple heuristics for determining diff type
	content = strings.ToLower(content)
	
	breakingKeywords := []string{
		"breaking", "urgent", "alert", "emergency", "crisis",
		"confirmed", "official", "announced", "declared",
	}

	for _, keyword := range breakingKeywords {
		if strings.Contains(content, keyword) {
			return "breaking"
		}
	}

	return "follow_up"
}

// GetStoryDiffStats returns statistics about story changes
func (d *DiffEngine) GetStoryDiffStats(storyID string) (map[string]interface{}, error) {
	// This could be expanded to include more detailed diff statistics
	hash, err := d.redis.GetStoryHash(storyID)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"current_hash":    hash,
		"last_updated":    time.Now(),
		"has_baseline":    hash != "",
	}, nil
}