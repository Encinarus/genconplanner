package api

import (
	"database/sql"

	firebase "firebase.google.com/go"
	"github.com/Encinarus/genconplanner/internal/background"
	"github.com/gin-gonic/gin"
)

func BuildAPIRoutes(api_group *gin.RouterGroup, db *sql.DB, gameCache *background.GameCache, app *firebase.App) {
	categoryRoutes(api_group, db)
	eventRoutes(api_group, db, gameCache)
	userRoutes(api_group, db, app)
}
