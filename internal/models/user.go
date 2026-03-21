package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UserRole defines the type of user
type UserRole string

const (
	RoleCorrespondent UserRole = "correspondent"
	RoleEditor        UserRole = "editor"
	RoleAdmin         UserRole = "admin"
)

// User represents a user in the system
type User struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	GoogleID      string             `bson:"google_id" json:"google_id"`
	Email         string             `bson:"email" json:"email"`
	Name          string             `bson:"name" json:"name"`
	Picture       string             `bson:"picture" json:"picture"`
	Role          UserRole           `bson:"role" json:"role"`
	Active        bool               `bson:"active" json:"active"`
	CreatedAt     time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time          `bson:"updated_at" json:"updated_at"`
	LastLoginAt   time.Time          `bson:"last_login_at" json:"last_login_at"`
}

// RSSFeed represents an RSS feed source
type RSSFeed struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name      string             `bson:"name" json:"name"`
	URL       string             `bson:"url" json:"url"`
	Category  string             `bson:"category" json:"category"`
	Active    bool               `bson:"active" json:"active"`
	AddedBy   string             `bson:"added_by,omitempty" json:"added_by,omitempty"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}
