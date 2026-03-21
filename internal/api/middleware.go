package api

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

var (
	rateLimiters = make(map[string]*rate.Limiter)
	mu           sync.RWMutex
)

// RateLimitMiddleware implements per-IP rate limiting
func RateLimitMiddleware(requestsPerSecond int) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		
		mu.RLock()
		limiter, exists := rateLimiters[ip]
		mu.RUnlock()

		if !exists {
			mu.Lock()
			// Double-check pattern
			if limiter, exists = rateLimiters[ip]; !exists {
				limiter = rate.NewLimiter(rate.Limit(requestsPerSecond), requestsPerSecond*2)
				rateLimiters[ip] = limiter
			}
			mu.Unlock()
		}

		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate limit exceeded",
				"message": "Too many requests from this IP",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// CORSMiddleware handles Cross-Origin Resource Sharing
func CORSMiddleware() gin.HandlerFunc {
	allowedOrigins := map[string]bool{
		"https://www.app-followup.duckdns.org": true,
		"http://localhost:4000":                true,
		"http://localhost:3000":                true,
	}

	// Also allow any origin set via env var
	if origin := os.Getenv("ALLOWED_ORIGINS"); origin != "" {
		for _, o := range strings.Split(origin, ",") {
			allowedOrigins[strings.TrimSpace(o)] = true
		}
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if allowedOrigins[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
		} else {
			c.Header("Access-Control-Allow-Origin", "*")
		}
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-User-ID, X-User-Role")
		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")
		c.Header("Cross-Origin-Opener-Policy", "same-origin-allow-popups")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// HealthCheck provides a simple health check endpoint
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now(),
		"service":   "followupmedium-newsroom",
	})
}

// LoggingMiddleware provides structured logging for requests
func LoggingMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf(`{"time":"%s","method":"%s","path":"%s","status":%d,"latency":"%s","ip":"%s","user_agent":"%s"}%s`,
			param.TimeStamp.Format(time.RFC3339),
			param.Method,
			param.Path,
			param.StatusCode,
			param.Latency,
			param.ClientIP,
			param.Request.UserAgent(),
			"\n",
		)
	})
}

// Cleanup function to remove old rate limiters (call periodically)
func CleanupRateLimiters() {
	ticker := time.NewTicker(time.Hour)
	go func() {
		for range ticker.C {
			mu.Lock()
			// In a production system, you'd implement proper cleanup logic
			// For now, we'll just clear all limiters periodically
			if len(rateLimiters) > 1000 {
				rateLimiters = make(map[string]*rate.Limiter)
			}
			mu.Unlock()
		}
	}()
}