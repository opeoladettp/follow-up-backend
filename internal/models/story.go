package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Story struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Title       string             `bson:"title" json:"title"`
	Description string             `bson:"description" json:"description"`
	Category    string             `bson:"category" json:"category"`
	Tags        []string           `bson:"tags" json:"tags"`
	Sources     []Source           `bson:"sources" json:"sources"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
	Status      string             `bson:"status" json:"status"` // active, archived, breaking
}

type Source struct {
	Type string `bson:"type" json:"type"` // rss, twitter
	URL  string `bson:"url" json:"url"`
	Name string `bson:"name" json:"name"`
}

type StoryLifecycle struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	StoryID      primitive.ObjectID `bson:"story_id" json:"story_id"`
	Developments []Development      `bson:"developments" json:"developments"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updated_at"`
}

type Development struct {
	ID          string            `bson:"id" json:"id"`
	Content     string            `bson:"content" json:"content"`
	Source      Source            `bson:"source" json:"source"`
	ContentHash string            `bson:"content_hash" json:"content_hash"`
	Timestamp   time.Time         `bson:"timestamp" json:"timestamp"`
	Type        string            `bson:"type" json:"type"` // breaking, follow_up, update
	Metadata    map[string]string `bson:"metadata" json:"metadata"`
	AIAnalysis  *AIAnalysis       `bson:"ai_analysis,omitempty" json:"ai_analysis,omitempty"`
}

type AIAnalysis struct {
	Sentiment    string  `bson:"sentiment" json:"sentiment"`
	Importance   int     `bson:"importance" json:"importance"` // 1-10 scale
	Summary      string  `bson:"summary" json:"summary"`
	KeyPoints    []string `bson:"key_points" json:"key_points"`
	VideoScript  string  `bson:"video_script,omitempty" json:"video_script,omitempty"`
	GeneratedAt  time.Time `bson:"generated_at" json:"generated_at"`
}

type TelemetryEvent struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	EventType string             `bson:"event_type" json:"event_type"`
	StoryID   *primitive.ObjectID `bson:"story_id,omitempty" json:"story_id,omitempty"`
	Success   bool               `bson:"success" json:"success"`
	Latency   int64              `bson:"latency_ms" json:"latency_ms"`
	Error     string             `bson:"error,omitempty" json:"error,omitempty"`
	Metadata  map[string]interface{} `bson:"metadata" json:"metadata"`
	Timestamp time.Time          `bson:"timestamp" json:"timestamp"`
}

type KPIMetrics struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Date            time.Time          `bson:"date" json:"date"`
	StoriesTracked  int                `bson:"stories_tracked" json:"stories_tracked"`
	DevelopmentsAdded int              `bson:"developments_added" json:"developments_added"`
	AvgLatency      float64            `bson:"avg_latency_ms" json:"avg_latency_ms"`
	ErrorRate       float64            `bson:"error_rate" json:"error_rate"`
	VideoGenerated  int                `bson:"videos_generated" json:"videos_generated"`
	APICallsCount   map[string]int     `bson:"api_calls_count" json:"api_calls_count"`
}

// NewsReport represents a correspondent's edited news report
type NewsReport struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	HeadlineID  string             `bson:"headline_id" json:"headline_id"`
	Title       string             `bson:"title" json:"title"`
	Script      string             `bson:"script" json:"script"`
	Author      string             `bson:"author" json:"author"`
	AvatarURL   string             `bson:"avatar_url,omitempty" json:"avatar_url,omitempty"`
	VideoJobID  string             `bson:"video_job_id,omitempty" json:"video_job_id,omitempty"`
	VideoStatus string             `bson:"video_status,omitempty" json:"video_status,omitempty"` // processing, completed, failed
	VideoURL    string             `bson:"video_url,omitempty" json:"video_url,omitempty"`
	Status      string             `bson:"status" json:"status"` // draft, generating, completed
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}
