package services

import (
	"context"
	"errors"
	"time"

	"followupmedium-newsroom/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type AuthService struct {
	db *mongo.Database
}

func NewAuthService(db *mongo.Database) *AuthService {
	return &AuthService{db: db}
}

// AuthenticateOrCreateUser handles Google OAuth authentication
func (s *AuthService) AuthenticateOrCreateUser(googleID, email, name, picture string) (*models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := s.db.Collection("users")

	// Try to find existing user
	var user models.User
	err := collection.FindOne(ctx, bson.M{"google_id": googleID}).Decode(&user)
	
	if err == mongo.ErrNoDocuments {
		// Create new user with correspondent role by default
		user = models.User{
			ID:          primitive.NewObjectID(),
			GoogleID:    googleID,
			Email:       email,
			Name:        name,
			Picture:     picture,
			Role:        models.RoleCorrespondent,
			Active:      true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			LastLoginAt: time.Now(),
		}

		_, err = collection.InsertOne(ctx, user)
		if err != nil {
			return nil, err
		}
		return &user, nil
	}

	if err != nil {
		return nil, err
	}

	// Update last login
	update := bson.M{
		"$set": bson.M{
			"last_login_at": time.Now(),
			"picture":       picture,
			"name":          name,
		},
	}
	_, err = collection.UpdateOne(ctx, bson.M{"_id": user.ID}, update)
	if err != nil {
		return nil, err
	}

	user.LastLoginAt = time.Now()
	return &user, nil
}

// GetUserByID retrieves a user by their ID
func (s *AuthService) GetUserByID(userID string) (*models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, err
	}

	var user models.User
	err = s.db.Collection("users").FindOne(ctx, bson.M{"_id": objID}).Decode(&user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// UpdateUserHeygenSettings saves a user's HeyGen avatar and voice IDs.
func (s *AuthService) UpdateUserHeygenSettings(userID, avatarID, voiceID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{
			"heygen_avatar_id": avatarID,
			"heygen_voice_id":  voiceID,
			"updated_at":       time.Now(),
		},
	}

	result, err := s.db.Collection("users").UpdateOne(ctx, bson.M{"_id": objID}, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errors.New("user not found")
	}
	return nil
}

// UpdateUserRole updates a user's role (admin only)
func (s *AuthService) UpdateUserRole(userID string, newRole models.UserRole) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{
			"role":       newRole,
			"updated_at": time.Now(),
		},
	}

	result, err := s.db.Collection("users").UpdateOne(ctx, bson.M{"_id": objID}, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("user not found")
	}

	return nil
}

// GetAllUsers retrieves all users (admin only)
func (s *AuthService) GetAllUsers() ([]models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := s.db.Collection("users").Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []models.User
	if err = cursor.All(ctx, &users); err != nil {
		return nil, err
	}

	return users, nil
}
