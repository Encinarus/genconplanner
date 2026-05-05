package api

import (

	firebase "firebase.google.com/go"
	"github.com/Encinarus/genconplanner/internal/background"
	"github.com/gin-gonic/gin"
)

func BuildAPIRoutes(api_group *gin.RouterGroup, repo EventRepository, gameCache *background.GameCache, app *firebase.App) {
	server := &Server{
		Repo:  repo,
		Auth:  &FirebaseAuthWrapper{App: app},
		Games: &GameCacheWrapper{Cache: gameCache},
	}

	categoryRoutes(api_group, server.Repo)
	eventRoutes(api_group, server.Repo, gameCache)

	authGroup := api_group.Group("/")
	authGroup.Use(server.AuthMiddleware())
	userRoutes(authGroup, server.Repo)
}
