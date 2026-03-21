package services

import (
	"context"
	"fmt"
	"time"

	"followupmedium-newsroom/internal/database"
	"followupmedium-newsroom/internal/models"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type StoryService struct {
	db    *database.MongoDB
	redis *database.Redis
}

func NewStoryService(db *database.MongoDB, redis *database.Redis) *StoryService {
	return &StoryService{
		db:    db,
		redis: redis,
	}
}

// CreateStory creates a new story
func (s *StoryService) CreateStory(story *models.Story) error {
	story.ID = primitive.NewObjectID()
	story.CreatedAt = time.Now()
	story.UpdatedAt = time.Now()
	story.Status = "active"

	_, err := s.db.Stories().InsertOne(context.Background(), story)
	if err != nil {
		s.logTelemetry("create_story", nil, false, err.Error(), 0)
		return fmt.Errorf("failed to create story: %w", err)
	}

	// Initialize story lifecycle
	lifecycle := &models.StoryLifecycle{
		ID:           primitive.NewObjectID(),
		StoryID:      story.ID,
		Developments: []models.Development{},
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	_, err = s.db.StoryLifecycles().InsertOne(context.Background(), lifecycle)
	if err != nil {
		logrus.WithError(err).Error("Failed to create story lifecycle")
	}

	s.logTelemetry("create_story", &story.ID, true, "", 0)
	return nil
}

// GetStoryContext retrieves full story timeline for MCP
func (s *StoryService) GetStoryContext(storyID string) (*models.StoryLifecycle, error) {
	startTime := time.Now()
	
	// Try cache first
	if cached, err := s.redis.GetCachedStoryContext(storyID); err == nil && cached != "" {
		logrus.Debug("Story context served from cache")
		// In a real implementation, you'd unmarshal the cached data
	}

	objectID, err := primitive.ObjectIDFromHex(storyID)
	if err != nil {
		return nil, fmt.Errorf("invalid story ID: %w", err)
	}

	var lifecycle models.StoryLifecycle
	err = s.db.StoryLifecycles().FindOne(
		context.Background(),
		bson.M{"story_id": objectID},
	).Decode(&lifecycle)

	latency := time.Since(startTime).Milliseconds()
	
	if err != nil {
		s.logTelemetry("get_story_context", &objectID, false, err.Error(), latency)
		return nil, fmt.Errorf("failed to get story context: %w", err)
	}

	// Cache the result
	s.redis.CacheStoryContext(storyID, lifecycle, 10*time.Minute)
	
	s.logTelemetry("get_story_context", &objectID, true, "", latency)
	return &lifecycle, nil
}

// AddDevelopment adds a new development to a story lifecycle
func (s *StoryService) AddDevelopment(storyID string, development models.Development) error {
	startTime := time.Now()
	
	objectID, err := primitive.ObjectIDFromHex(storyID)
	if err != nil {
		return fmt.Errorf("invalid story ID: %w", err)
	}

	development.ID = primitive.NewObjectID().Hex()
	development.Timestamp = time.Now()

	update := bson.M{
		"$push": bson.M{"developments": development},
		"$set":  bson.M{"updated_at": time.Now()},
	}

	result, err := s.db.StoryLifecycles().UpdateOne(
		context.Background(),
		bson.M{"story_id": objectID},
		update,
	)

	latency := time.Since(startTime).Milliseconds()

	if err != nil {
		s.logTelemetry("add_development", &objectID, false, err.Error(), latency)
		return fmt.Errorf("failed to add development: %w", err)
	}

	if result.MatchedCount == 0 {
		s.logTelemetry("add_development", &objectID, false, "story not found", latency)
		return fmt.Errorf("story not found")
	}

	// Invalidate cache
	s.redis.Client.Del(context.Background(), "context:"+storyID)

	s.logTelemetry("add_development", &objectID, true, "", latency)
	logrus.WithFields(logrus.Fields{
		"story_id":        storyID,
		"development_id":  development.ID,
		"type":           development.Type,
	}).Info("Development added to story")

	return nil
}

// GetActiveStories returns all active stories
func (s *StoryService) GetActiveStories() ([]models.Story, error) {
	cursor, err := s.db.Stories().Find(
		context.Background(),
		bson.M{"status": "active"},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get active stories: %w", err)
	}
	defer cursor.Close(context.Background())

	var stories []models.Story
	if err := cursor.All(context.Background(), &stories); err != nil {
		return nil, fmt.Errorf("failed to decode stories: %w", err)
	}

	return stories, nil
}

// UpdateKPIDashboard pushes metrics to dashboard collection
func (s *StoryService) UpdateKPIDashboard() error {
	startTime := time.Now()
	
	// Aggregate today's metrics
	today := time.Now().Truncate(24 * time.Hour)
	
	// Count stories tracked today
	storiesCount, err := s.db.Stories().CountDocuments(
		context.Background(),
		bson.M{"created_at": bson.M{"$gte": today}},
	)
	if err != nil {
		return fmt.Errorf("failed to count stories: %w", err)
	}

	// Count developments added today
	pipeline := mongo.Pipeline{
		{{"$match", bson.D{{"updated_at", bson.D{{"$gte", today}}}}}},
		{{"$unwind", "$developments"}},
		{{"$match", bson.D{{"developments.timestamp", bson.D{{"$gte", today}}}}}},
		{{"$count", "total"}},
	}

	cursor, err := s.db.StoryLifecycles().Aggregate(context.Background(), pipeline)
	if err != nil {
		return fmt.Errorf("failed to aggregate developments: %w", err)
	}
	defer cursor.Close(context.Background())

	var developmentCount struct {
		Total int `bson:"total"`
	}
	if cursor.Next(context.Background()) {
		cursor.Decode(&developmentCount)
	}

	// Calculate average latency from telemetry
	avgLatency, errorRate := s.calculateTelemetryMetrics(today)

	metrics := models.KPIMetrics{
		ID:                primitive.NewObjectID(),
		Date:              today,
		StoriesTracked:    int(storiesCount),
		DevelopmentsAdded: developmentCount.Total,
		AvgLatency:        avgLatency,
		ErrorRate:         errorRate,
		VideoGenerated:    0, // Will be updated by AI service
		APICallsCount:     make(map[string]int),
	}

	_, err = s.db.KPIDashboard().InsertOne(context.Background(), metrics)
	latency := time.Since(startTime).Milliseconds()
	
	if err != nil {
		s.logTelemetry("update_kpi_dashboard", nil, false, err.Error(), latency)
		return fmt.Errorf("failed to update KPI dashboard: %w", err)
	}

	s.logTelemetry("update_kpi_dashboard", nil, true, "", latency)
	return nil
}

func (s *StoryService) calculateTelemetryMetrics(since time.Time) (float64, float64) {
	// Aggregate telemetry data for metrics calculation
	pipeline := mongo.Pipeline{
		{{"$match", bson.D{{"timestamp", bson.D{{"$gte", since}}}}}},
		{{"$group", bson.D{
			{"_id", nil},
			{"avg_latency", bson.D{{"$avg", "$latency_ms"}}},
			{"total_events", bson.D{{"$sum", 1}}},
			{"error_events", bson.D{{"$sum", bson.D{{"$cond", bson.A{bson.D{{"$eq", bson.A{"$success", false}}}, 1, 0}}}}}},
		}}},
	}

	cursor, err := s.db.Telemetry().Aggregate(context.Background(), pipeline)
	if err != nil {
		logrus.WithError(err).Error("Failed to calculate telemetry metrics")
		return 0, 0
	}
	defer cursor.Close(context.Background())

	var result struct {
		AvgLatency  float64 `bson:"avg_latency"`
		TotalEvents int     `bson:"total_events"`
		ErrorEvents int     `bson:"error_events"`
	}

	if cursor.Next(context.Background()) {
		cursor.Decode(&result)
	}

	errorRate := 0.0
	if result.TotalEvents > 0 {
		errorRate = float64(result.ErrorEvents) / float64(result.TotalEvents) * 100
	}

	return result.AvgLatency, errorRate
}

func (s *StoryService) logTelemetry(eventType string, storyID *primitive.ObjectID, success bool, errorMsg string, latency int64) {
	event := models.TelemetryEvent{
		ID:        primitive.NewObjectID(),
		EventType: eventType,
		StoryID:   storyID,
		Success:   success,
		Latency:   latency,
		Error:     errorMsg,
		Metadata:  make(map[string]interface{}),
		Timestamp: time.Now(),
	}

	_, err := s.db.Telemetry().InsertOne(context.Background(), event)
	if err != nil {
		logrus.WithError(err).Error("Failed to log telemetry event")
	}
}