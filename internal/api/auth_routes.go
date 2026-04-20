package api

import (
	"net/http"

	"followupmedium-newsroom/internal/models"
	"followupmedium-newsroom/internal/services"

	"github.com/gin-gonic/gin"
)

func SetupAuthRoutes(router *gin.RouterGroup, authService *services.AuthService) {
	auth := router.Group("/auth")
	{
		auth.POST("/google", googleAuth(authService))
		auth.GET("/me", requireAuth(), getCurrentUser(authService))
		auth.POST("/logout", requireAuth(), logout())
		auth.PUT("/me/heygen", requireAuth(), updateHeygenSettings(authService))
	}

	// Admin routes
	admin := router.Group("/admin")
	admin.Use(requireAuth(), requireRole(models.RoleAdmin))
	{
		admin.GET("/users", getAllUsers(authService))
		admin.PUT("/users/:id/role", updateUserRole(authService))
	}
}

func googleAuth(authService *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			GoogleID string `json:"google_id" binding:"required"`
			Email    string `json:"email" binding:"required"`
			Name     string `json:"name" binding:"required"`
			Picture  string `json:"picture"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		user, err := authService.AuthenticateOrCreateUser(req.GoogleID, req.Email, req.Name, req.Picture)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Authentication failed"})
			return
		}

		// In production, generate JWT token here
		c.JSON(http.StatusOK, gin.H{
			"user":    user,
			"message": "Authentication successful",
		})
	}
}

func getCurrentUser(authService *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		user, err := authService.GetUserByID(userID.(string))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"user": user})
	}
}

func logout() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Clear session/token
		c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
	}
}

func getAllUsers(authService *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		users, err := authService.GetAllUsers()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"users": users})
	}
}

func updateUserRole(authService *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Param("id")
		
		var req struct {
			Role models.UserRole `json:"role" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := authService.UpdateUserRole(userID, req.Role); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "User role updated successfully"})
	}
}

// Middleware functions
func requireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// In production, validate JWT token here
		// For now, we'll use a simple header check
		userID := c.GetHeader("X-User-ID")
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		c.Set("userID", userID)
		c.Next()
	}
}

func requireRole(requiredRole models.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole := c.GetHeader("X-User-Role")
		if models.UserRole(userRole) != requiredRole {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}

func updateHeygenSettings(authService *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		var req struct {
			AvatarID string `json:"heygen_avatar_id"`
			VoiceID  string `json:"heygen_voice_id"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := authService.UpdateUserHeygenSettings(userID.(string), req.AvatarID, req.VoiceID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "HeyGen settings saved"})
	}
}
