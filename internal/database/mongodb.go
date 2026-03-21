package database

import (
	"context"
	"net/url"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDB struct {
	Client   *mongo.Client
	Database *mongo.Database
}

func NewMongoDB(uri string) (*MongoDB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	// Extract database name from URI path (e.g. .../newsroom?...)
	dbName := "newsroom"
	if parsed, err := url.Parse(uri); err == nil {
		if path := strings.TrimPrefix(parsed.Path, "/"); path != "" {
			dbName = path
		}
	}

	return &MongoDB{
		Client:   client,
		Database: client.Database(dbName),
	}, nil
}

func (m *MongoDB) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return m.Client.Disconnect(ctx)
}

func (m *MongoDB) Stories() *mongo.Collection {
	return m.Database.Collection("stories")
}

func (m *MongoDB) StoryLifecycles() *mongo.Collection {
	return m.Database.Collection("story_lifecycles")
}

func (m *MongoDB) Telemetry() *mongo.Collection {
	return m.Database.Collection("telemetry")
}

func (m *MongoDB) KPIDashboard() *mongo.Collection {
	return m.Database.Collection("kpi_dashboard")
}

func (m *MongoDB) NewsReports() *mongo.Collection {
	return m.Database.Collection("news_reports")
}
