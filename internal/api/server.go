package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	firebase "firebase.google.com/go"
	"github.com/Encinarus/genconplanner/internal/background"
	"github.com/Encinarus/genconplanner/internal/locations"
	"github.com/Encinarus/genconplanner/internal/logging"
	"github.com/Encinarus/genconplanner/internal/postgres"
	"github.com/gin-gonic/gin"
)

const userEmailKey = "userEmail"

type Server struct {
	Repo            EventRepository
	Auth            AuthService
	Games           GameService
	LocationMatcher *locations.Matcher
}

func NewServer(repo EventRepository, auth AuthService, games GameService) *Server {
	matcher := locations.NewMatcher()

	// Load location pins directly from PostgreSQL table public.gencon_locations
	if pgRepo, ok := repo.(*PostgresRepository); ok && pgRepo != nil && pgRepo.DB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := matcher.LoadFromDB(ctx, pgRepo.DB); err != nil {
			log.Printf("Error loading locations from database: %v", err)
		}
	}

	return &Server{
		Repo:            repo,
		Auth:            auth,
		Games:           games,
		LocationMatcher: matcher,
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
		idToken := c.GetHeader("Authorization")
		if len(idToken) > 7 && idToken[:7] == "Bearer " {
			idToken = idToken[7:]
		}
		if idToken == "" && !strings.HasPrefix(c.Request.URL.Path, "/api/v1/") {
			idToken, _ = c.Cookie("signinToken")
		}

		if idToken == "" {
			logging.LogCtx(c, "AuthMiddleware: no token found in cookie or header")
			c.AbortWithStatusJSON(401, ErrorResponse{Error: "Unauthorized"})
			return
		}

		email, err := s.Auth.VerifyIDToken(c.Request.Context(), idToken)
		if err != nil {
			logging.LogCtx(c, "AuthMiddleware: token verification failed: %v", err)
			c.AbortWithStatusJSON(401, ErrorResponse{Error: "Unauthorized"})
			return
		}
		if email == "" {
			logging.LogCtx(c, "AuthMiddleware: token verified but email is empty")
			c.AbortWithStatusJSON(401, ErrorResponse{Error: "Unauthorized"})
			return
		}

		logging.LogCtx(c, "AuthMiddleware: authenticated user %s", email)
		c.Set(userEmailKey, strings.ToLower(email))
		c.Next()
	}
}

func (s *Server) AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		email := GetUserEmail(c)
		if email == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
			return
		}
		isAdmin, err := s.Repo.IsAdmin(email)
		if err != nil {
			log.Printf("AdminOnly error checking admin: %v", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
			return
		}
		if !isAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, ErrorResponse{Error: "Forbidden"})
			return
		}
		c.Next()
	}
}

func (s *Server) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		idToken := c.GetHeader("Authorization")
		if len(idToken) > 7 && idToken[:7] == "Bearer " {
			idToken = idToken[7:]
		}
		if idToken == "" && !strings.HasPrefix(c.Request.URL.Path, "/api/v1/") {
			idToken, _ = c.Cookie("signinToken")
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

		// Admin protected group
		admin := v1.Group("/admin")
		admin.Use(s.AuthMiddleware(), s.AdminOnly())
		{
			admin.GET("/orgs", s.ViewOrgs)
			admin.POST("/orgs/merge", s.MergeOrgs)
			admin.GET("/orgs/merge-suggestions", s.GetMergeSuggestions)
		}
	}
}

func (s *Server) MatchLocation(location, roomName, tableNumber string) string {
	if s == nil || s.LocationMatcher == nil {
		return ""
	}
	return s.LocationMatcher.MatchLocation(location, roomName, tableNumber)
}

func (s *Server) GetUpdateStatus(c *gin.Context) {
	lastUpdate, err := s.Repo.GetLastUpdate()
	if err != nil {
		log.Printf("Error getting last update: %v\n", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"lastUpdate": lastUpdate})
}
