package web

import (
	"database/sql"
	"github.com/Encinarus/genconplanner/internal/postgres"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"strconv"
)

func MergeOrgs(db *sql.DB) gin.HandlerFunc {
	return func (c *gin.Context) {
		// TODO(alek): add acl check here. I guess I need a concept of admin users
		stringOrgIds, ok := c.GetPostFormArray("id")
		if !ok {
			log.Printf("Unable to get array")
			return
		}

		orgIds := make([]int64, 0, len(stringOrgIds))
		for _, stringId := range stringOrgIds {
			id, err := strconv.ParseInt(stringId, 10, 64)
			if err == nil {
				orgIds = append(orgIds, id)
			} else {
				log.Printf("Couldn't parse %s", stringId)
			}
		}
		postgres.MergeOrgs(db, orgIds)

		orgs, err := postgres.LoadAllOrgs(db)
		if err != nil {
			c.Error(err)
			return
		}
		c.HTML(http.StatusOK, "organizers.html", gin.H{
			"orgs": orgs,
		})
	}
}

func ViewOrgs(db *sql.DB) gin.HandlerFunc {
	return func (c *gin.Context) {
		// TODO(alek): add acl check here. I guess I need a concept of admin users
		orgs, err := postgres.LoadAllOrgs(db)
		if err != nil {
			c.Error(err)
			return
		}
		//c.Header("Content-Type", "application/json")
		//json.NewEncoder(c.Writer).Encode(orgs)

		c.HTML(http.StatusOK, "organizers.html", gin.H{
			"orgs": orgs,
		})
	}
}