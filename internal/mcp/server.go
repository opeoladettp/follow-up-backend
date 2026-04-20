package mcp

import (
	"fmt"
	"net/http"
	"time"

	"followupmedium-newsroom/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type Server struct {
	storyService *services.StoryService
	aiService    *services.AIService
	router       *gin.Engine
}

type MCPRequest struct {
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params"`
	ID     string                 `json:"id"`
}

type MCPResponse struct {
	Result interface{} `json:"result,omitempty"`
	Error  *MCPError   `json:"error,omitempty"`
	ID     string      `json:"id"`
}

type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func NewServer(storyService *services.StoryService, aiService *services.AIService) *Server {
	server := &Server{
		storyService: storyService,
		aiService:    aiService,
	}
	
	server.setupRouter()
	return server
}

func (s *Server) setupRouter() {
	s.router = gin.New()
	s.router.Use(gin.Logger())
	s.router.Use(gin.Recovery())
	
	// MCP protocol endpoints
	s.router.POST("/mcp", s.handleMCPRequest)
	s.router.GET("/mcp/tools", s.listTools)
	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "mcp-server"})
	})
}

func (s *Server) Start(port int) {
	logrus.Infof("Starting MCP server on port %d", port)
	if err := s.router.Run(fmt.Sprintf(":%d", port)); err != nil {
		logrus.WithError(err).Fatal("Failed to start MCP server")
	}
}

func (s *Server) handleMCPRequest(c *gin.Context) {
	var request MCPRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		s.sendError(c, request.ID, -32700, "Parse error")
		return
	}

	logrus.WithFields(logrus.Fields{
		"method": request.Method,
		"id":     request.ID,
	}).Debug("Handling MCP request")

	var result interface{}
	var err error

	switch request.Method {
	case "get_story_context":
		result, err = s.getStoryContext(request.Params)
	case "trigger_production_pipeline":
		result, err = s.triggerProductionPipeline(request.Params)
	case "update_kpi_dashboard":
		result, err = s.updateKPIDashboard(request.Params)
	default:
		s.sendError(c, request.ID, -32601, "Method not found")
		return
	}

	if err != nil {
		logrus.WithError(err).Error("MCP method execution failed")
		s.sendError(c, request.ID, -32603, err.Error())
		return
	}

	response := MCPResponse{
		Result: result,
		ID:     request.ID,
	}

	c.JSON(http.StatusOK, response)
}

func (s *Server) getStoryContext(params map[string]interface{}) (interface{}, error) {
	storyID, ok := params["story_id"].(string)
	if !ok {
		return nil, fmt.Errorf("story_id parameter is required")
	}

	startTime := time.Now()
	lifecycle, err := s.storyService.GetStoryContext(storyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get story context: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"story_id":     storyID,
		"developments": len(lifecycle.Developments),
		"latency":      time.Since(startTime).Milliseconds(),
	}).Info("Story context retrieved via MCP")

	return map[string]interface{}{
		"story_id":     lifecycle.StoryID.Hex(),
		"developments": lifecycle.Developments,
		"created_at":   lifecycle.CreatedAt,
		"updated_at":   lifecycle.UpdatedAt,
		"total_count":  len(lifecycle.Developments),
	}, nil
}

func (s *Server) triggerProductionPipeline(params map[string]interface{}) (interface{}, error) {
	scriptText, ok := params["script_text"].(string)
	if !ok {
		return nil, fmt.Errorf("script_text parameter is required")
	}

	identityImage, _ := params["identity_image"].(string)
	if identityImage == "" {
		// Use default identity image from config
		identityImage = "default_avatar.jpg" // This should come from config
	}

	reportID, _ := params["report_id"].(string)
	if reportID == "" {
		reportID = "mcp-" + fmt.Sprintf("%d", time.Now().Unix())
	}

	startTime := time.Now()
	videoJobID, err := s.aiService.TriggerProductionPipeline(scriptText, identityImage, reportID, "", "", "")
	if err != nil {
		return nil, fmt.Errorf("failed to trigger production pipeline: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"script_length":  len(scriptText),
		"identity_image": identityImage,
		"report_id":      reportID,
		"video_job_id":   videoJobID,
		"latency":        time.Since(startTime).Milliseconds(),
	}).Info("Production pipeline triggered via MCP")

	return map[string]interface{}{
		"status":         "triggered",
		"video_job_id":   videoJobID,
		"script_length":  len(scriptText),
		"identity_image": identityImage,
		"timestamp":      time.Now(),
	}, nil
}

func (s *Server) updateKPIDashboard(params map[string]interface{}) (interface{}, error) {
	startTime := time.Now()
	err := s.storyService.UpdateKPIDashboard()
	if err != nil {
		return nil, fmt.Errorf("failed to update KPI dashboard: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"latency": time.Since(startTime).Milliseconds(),
	}).Info("KPI dashboard updated via MCP")

	return map[string]interface{}{
		"status":    "updated",
		"timestamp": time.Now(),
	}, nil
}

func (s *Server) listTools(c *gin.Context) {
	tools := []map[string]interface{}{
		{
			"name":        "get_story_context",
			"description": "Retrieves the full timeline and context for a specific story",
			"parameters": map[string]interface{}{
				"story_id": map[string]interface{}{
					"type":        "string",
					"description": "The unique identifier of the story",
					"required":    true,
				},
			},
		},
		{
			"name":        "trigger_production_pipeline",
			"description": "Triggers video generation pipeline with Veo 3.1 and Nano Banana avatar overlay",
			"parameters": map[string]interface{}{
				"script_text": map[string]interface{}{
					"type":        "string",
					"description": "The script text for video generation",
					"required":    true,
				},
				"identity_image": map[string]interface{}{
					"type":        "string",
					"description": "URL to the identity image for avatar consistency",
					"required":    false,
				},
			},
		},
		{
			"name":        "update_kpi_dashboard",
			"description": "Updates the KPI dashboard with latest metrics for Looker Studio",
			"parameters":  map[string]interface{}{},
		},
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"tools": tools,
	})
}

func (s *Server) sendError(c *gin.Context, id string, code int, message string) {
	response := MCPResponse{
		Error: &MCPError{
			Code:    code,
			Message: message,
		},
		ID: id,
	}
	c.JSON(http.StatusOK, response)
}