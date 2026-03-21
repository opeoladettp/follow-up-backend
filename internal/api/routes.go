package api

import (
	"net/http"
	"time"

	"followupmedium-newsroom/internal/models"
	"followupmedium-newsroom/internal/services"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(router *gin.RouterGroup, storyService *services.StoryService, diffEngine *services.DiffEngine) {
	// Story management routes
	stories := router.Group("/stories")
	{
		stories.GET("/", getActiveStories(storyService))
		stories.POST("/", createStory(storyService))
		stories.GET("/:id/context", getStoryContext(storyService))
		stories.GET("/:id/diff-stats", getStoryDiffStats(diffEngine))
	}

	// Health and metrics routes
	router.GET("/metrics", getMetrics(storyService))
	router.POST("/kpi/update", updateKPIDashboard(storyService))
}

func getActiveStories(storyService *services.StoryService) gin.HandlerFunc {
	return func(c *gin.Context) {
		stories, err := storyService.GetActiveStories()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"stories": stories,
			"count":   len(stories),
		})
	}
}

func createStory(storyService *services.StoryService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var story models.Story
		if err := c.ShouldBindJSON(&story); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := storyService.CreateStory(&story); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message":  "Story created successfully",
			"story_id": story.ID.Hex(),
		})
	}
}

func getStoryContext(storyService *services.StoryService) gin.HandlerFunc {
	return func(c *gin.Context) {
		storyID := c.Param("id")
		
		lifecycle, err := storyService.GetStoryContext(storyID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, lifecycle)
	}
}

func getStoryDiffStats(diffEngine *services.DiffEngine) gin.HandlerFunc {
	return func(c *gin.Context) {
		storyID := c.Param("id")
		
		stats, err := diffEngine.GetStoryDiffStats(storyID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, stats)
	}
}

func getMetrics(storyService *services.StoryService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Return basic system metrics
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now(),
			"uptime":    time.Since(startTime).String(),
		})
	}
}

func updateKPIDashboard(storyService *services.StoryService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := storyService.UpdateKPIDashboard(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":   "KPI dashboard updated successfully",
			"timestamp": time.Now(),
		})
	}
}

var startTime = time.Now()