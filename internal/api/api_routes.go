package api

import (
	"database/sql"

	"github.com/Encinarus/genconplanner/internal/background"
	"github.com/gin-gonic/gin"
)

func BuildAPIRoutes(api_group *gin.RouterGroup, db *sql.DB, gameCache *background.GameCache) {
	categoryRoutes(api_group, db)
	eventRoutes(api_group, db, gameCache)
}
