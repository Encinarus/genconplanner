package api

import (
	"database/sql"

	"github.com/gin-gonic/gin"
)

func BuildAPIRoutes(api_group *gin.RouterGroup, db *sql.DB) {
	categoryRoutes(api_group, db)
}
