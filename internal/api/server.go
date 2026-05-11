package api

import (
	"context"
	"fmt"
	"log"
	"strings"
	"net/http"

	firebase "firebase.google.com/go"
	"github.com/Encinarus/genconplanner/internal/background"
	"github.com/Encinarus/genconplanner/internal/postgres"
	"github.com/gin-gonic/gin"
)

const userEmailKey = "userEmail"

type Server struct {
	Repo  EventRepository
	Auth  AuthService
	Games GameService
}

func NewServer(repo EventRepository, auth AuthService, games GameService) *Server {
	return &Server{
		Repo:  repo,
		Auth:  auth,
		Games: games,
	}
}

// FirebaseAuthWrapper implements AuthService using Firebase.
type FirebaseAuthWrapper struct {
	App *firebase.App
}

func (w *FirebaseAuthWrapper) VerifyIDToken(ctx context.Context, idToken string) (string, error) {
	if w.App == nil {
		log.Println("FirebaseAuthWrapper: App is nil")
		return "", fmt.Errorf("auth app not initialized")
	}
	client, err := w.App.Auth(context.Background())
	if err != nil {
		return "", err
	}
	token, err := client.VerifyIDToken(ctx, idToken)
	if err != nil {
		return "", err
	}
	if token != nil {
		if email, ok := token.Claims["email"].(string); ok {
			return email, nil
		}
	}
	return "", nil
}

// GameCacheWrapper implements GameService using the background GameCache.
type GameCacheWrapper struct {
	Cache *background.GameCache
}

func (w *GameCacheWrapper) FindGame(name string) *postgres.Game {
	return w.Cache.FindGame(name)
}

func (s *Server) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		idToken, err := c.Cookie("signinToken")
		if err != nil || idToken == "" {
			// Try header as fallback
			idToken = c.GetHeader("Authorization")
			if len(idToken) > 7 && idToken[:7] == "Bearer " {
				idToken = idToken[7:]
			}
		}

		if idToken == "" {
			log.Printf("AuthMiddleware: no token found in cookie or header\n")
			c.AbortWithStatusJSON(401, ErrorResponse{Error: "Unauthorized"})
			return
		}

		email, err := s.Auth.VerifyIDToken(c.Request.Context(), idToken)
		if err != nil {
			log.Printf("AuthMiddleware: token verification failed: %v\n", err)
			c.AbortWithStatusJSON(401, ErrorResponse{Error: "Unauthorized"})
			return
		}
		if email == "" {
			log.Println("AuthMiddleware: token verified but email is empty")
			c.AbortWithStatusJSON(401, ErrorResponse{Error: "Unauthorized"})
			return
		}

		log.Printf("AuthMiddleware: authenticated user %s\n", email)
		c.Set(userEmailKey, strings.ToLower(email))
		c.Next()
	}
}

func (s *Server) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		idToken, err := c.Cookie("signinToken")
		if err != nil || idToken == "" {
			// Try header as fallback
			idToken = c.GetHeader("Authorization")
			if len(idToken) > 7 && idToken[:7] == "Bearer " {
				idToken = idToken[7:]
			}
		}

		if idToken == "" {
			c.Next()
			return
		}

		email, err := s.Auth.VerifyIDToken(c.Request.Context(), idToken)
		if err == nil && email != "" {
			c.Set(userEmailKey, strings.ToLower(email))
		}
		c.Next()
	}
}

func GetUserEmail(c *gin.Context) string {
	if email, ok := c.Get(userEmailKey); ok {
		return email.(string)
	}
	return ""
}

func (s *Server) RegisterRoutes(group *gin.RouterGroup) {
	v1 := group.Group("/v1")
	{
		v1.GET("/metadata/last_update", s.GetUpdateStatus)
		s.registerCategoryRoutes(v1)
		s.registerEventRoutes(v1)

		// Auth protected group
		auth := v1.Group("/")
		auth.Use(s.AuthMiddleware())
		{
			s.registerUserRoutes(auth)
		}
	}
}

func (s *Server) GetUpdateStatus(c *gin.Context) {
	lastUpdate, err := s.Repo.GetLastUpdate()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"lastUpdate": lastUpdate})
}
