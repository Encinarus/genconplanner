package api

import (
	"context"

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
	client, err := w.App.Auth(ctx)
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
		if err != nil {
			c.AbortWithStatusJSON(401, ErrorResponse{Error: "Unauthorized"})
			return
		}

		email, err := s.Auth.VerifyIDToken(c.Request.Context(), idToken)
		if err != nil || email == "" {
			c.AbortWithStatusJSON(401, ErrorResponse{Error: "Unauthorized"})
			return
		}

		c.Set(userEmailKey, email)
		c.Next()
	}
}

func (s *Server) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		idToken, err := c.Cookie("signinToken")
		if err != nil {
			c.Next()
			return
		}

		email, err := s.Auth.VerifyIDToken(c.Request.Context(), idToken)
		if err == nil && email != "" {
			c.Set(userEmailKey, email)
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
		v1.GET("/category/:year", s.ListCategories)
		v1.GET("/event/:event_id", s.LookupEvent)
		v1.GET("/events", s.SearchEvents)
		v1.POST("/events", s.SearchEvents)

		// Auth protected group
		auth := v1.Group("/")
		auth.Use(s.AuthMiddleware())
		{
			auth.GET("/user", s.GetUser)
			auth.GET("/user/events/:email/:year", s.LoadUserEvents)
		}
	}
}
