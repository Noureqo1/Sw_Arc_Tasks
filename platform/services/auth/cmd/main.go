package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"platform/auth/internal/db"
	"platform/auth/internal/tokens"
	"platform/auth/internal/users"

	"github.com/gin-gonic/gin"
)

func main() {
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/authdb?sslmode=disable"
	}
	database, err := db.Connect(dsn)
	if err != nil {
		log.Fatal("failed to connect DB:", err)
	}

	userService := users.NewUserService(database)
	tokenService := tokens.NewTokenService(
		database,
		os.Getenv("JWT_SECRET"),
		time.Minute*15,
		time.Hour*24*7,
	)

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "Auth service healthy\n")
	})

	r.POST("/register", func(c *gin.Context) {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		err := userService.RegisterUser(req.Email, req.Password)
		if err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "registered"})
	})

	r.POST("/login", func(c *gin.Context) {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		user, err := userService.LoginUser(req.Email, req.Password)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		accessToken, refreshToken, err := tokenService.GenerateTokens(user.ID, user.Roles)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store refresh token"})
			return
		}

		err = tokenService.StoreRefreshToken(user.ID, refreshToken)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store refresh token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
		})
	})

	r.POST("/refresh", func(c *gin.Context) {
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}

		userID, err := tokenService.ValidateRefreshToken(req.RefreshToken)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired refresh token"})
			return
		}

		// generate new tokens
		accessToken, newRefresh, err := tokenService.GenerateTokens(userID, []string{"user"})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "token generation failed"})
			return
		}

		// optionally store new refresh, invalidate the old one
		err = tokenService.StoreRefreshToken(userID, newRefresh)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store refresh token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"access_token":  accessToken,
			"refresh_token": newRefresh,
		})
	})

	addr := ":8081"
	if port := os.Getenv("AUTH_PORT"); port != "" {
		addr = ":" + port
	}

	log.Printf("Auth service listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal("Failed to start Auth service:", err)
	}
}
