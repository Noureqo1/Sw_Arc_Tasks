package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"platform/gateway/internal/auth"
	"platform/gateway/internal/limiter"
)

var (
	authServiceURL     string
	functionServiceURL string
)

func main() {
	authServiceURL = os.Getenv("AUTH_SERVICE_URL")
	if authServiceURL == "" {
		authServiceURL = "http://auth:8081"
	}

	functionServiceURL = os.Getenv("FUNCTION_SERVICE_URL")
	if functionServiceURL == "" {
		functionServiceURL = "http://functionservice:8082"
	}

	r := gin.Default()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     "redis:6379",
		Password: "",
		DB:       0,
	})

	r.Use(concurrencyMiddleware(redisClient))
	r.Use(rateLimitMiddleware(redisClient))

	public := r.Group("/auth")
	{
		public.POST("/register", forwardToAuthService)
		public.POST("/login", forwardToAuthService)
		public.POST("/refresh", forwardToAuthService)
	}

	protected := r.Group("/")
	protected.Use(auth.AuthMiddleware())
	{
		protected.GET("/health", func(c *gin.Context) {
			c.String(http.StatusOK, "API Gateway is healthy\n")
		})
		protected.Any("/functions", forwardToFunctionService)
		protected.Any("/functions/*rest", forwardToFunctionService)

		protected.Any("/jobs", forwardToFunctionService)
		protected.Any("/jobs/*rest", forwardToFunctionService)

		admin := protected.Group("/admin")
		admin.Use(auth.RequireRoles("admin"))
		{
			admin.GET("/dashboard", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"dashboard": "admin metrics"})
			})
		}
	}

	addr := ":8080"
	if port := os.Getenv("GATEWAY_PORT"); port != "" {
		addr = ":" + port
	}

	log.Printf("Starting gateway on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal("Failed to start gateway:", err)
	}
}

func concurrencyMiddleware(rdb *redis.Client) gin.HandlerFunc {
	return limiter.NewConcurrencyMiddleware(rdb, 5, 1*time.Minute)
}

func rateLimitMiddleware(rdb *redis.Client) gin.HandlerFunc {
	algo := os.Getenv("RATE_LIMIT_ALGO")
	if algo == "" {
		algo = "token-bucket"
	}
	limitMW, _ := limiter.NewRateLimitMiddleware(rdb, algo, 10, "1m")
	return limitMW
}

func forwardToAuthService(c *gin.Context) {
	path := strings.TrimPrefix(c.Request.URL.Path, "/auth")
	targetURL := fmt.Sprintf("%s%s", authServiceURL, path)
	log.Printf("Forwarding to Auth Service => %s", targetURL)

	req, err := http.NewRequest(c.Request.Method, targetURL, c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
		return
	}

	for k, v := range c.Request.Header {
		req.Header[k] = v
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "auth service unreachable"})
		return
	}
	defer resp.Body.Close()

	c.Status(resp.StatusCode)
	for k, v := range resp.Header {
		c.Writer.Header()[k] = v
	}
	bodyBytes, _ := io.ReadAll(resp.Body)
	c.Writer.Write(bodyBytes)
}

func forwardToFunctionService(c *gin.Context) {
	finalURL := functionServiceURL + c.Request.URL.Path

	if c.Request.URL.RawQuery != "" {
		finalURL += "?" + c.Request.URL.RawQuery
	}

	log.Printf("Forwarding to Function Service => %s", finalURL)

	req, err := http.NewRequest(c.Request.Method, finalURL, c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
		return
	}

	// Copy headers
	for k, v := range c.Request.Header {
		req.Header[k] = v
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "function service unreachable"})
		return
	}
	defer resp.Body.Close()

	c.Status(resp.StatusCode)
	for k, v := range resp.Header {
		c.Writer.Header()[k] = v
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	c.Writer.Write(bodyBytes)
}
