package api

import (

	firebase "firebase.google.com/go"
	"github.com/Encinarus/genconplanner/internal/background"
	"github.com/gin-gonic/gin"
)

func BuildAPIRoutes(api_group *gin.RouterGroup, repo EventRepository, gameCache *background.GameCache, app *firebase.App) {
	categoryRoutes(api_group, repo)
	eventRoutes(api_group, repo, gameCache)
	userRoutes(api_group, repo, app)
}
