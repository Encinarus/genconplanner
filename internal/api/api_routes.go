package api

import (
	firebase "firebase.google.com/go"
	"github.com/Encinarus/genconplanner/internal/background"
	"github.com/gin-gonic/gin"
)

func BuildAPIRoutes(api_group *gin.RouterGroup, repo EventRepository, gameCache *background.GameCache, app *firebase.App) {
	server := NewServer(
		repo,
		&FirebaseAuthWrapper{App: app},
		&GameCacheWrapper{Cache: gameCache},
	)

	server.RegisterRoutes(api_group)
}
