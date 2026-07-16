package api

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AdminOrganizer struct {
	Id        int64    `json:"id"`
	Aliases   []string `json:"aliases"`
	NumEvents int64    `json:"numEvents"`
}

type MergeOrgsRequest struct {
	Ids []int64 `json:"ids" binding:"required"`
}

func (s *Server) ViewOrgs(c *gin.Context) {
	orgs, err := s.Repo.LoadAllOrgs()
	if err != nil {
		log.Printf("ViewOrgs error: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	results := make([]AdminOrganizer, 0, len(orgs))
	for _, o := range orgs {
		if o != nil && len(o.Aliases) > 0 {
			results = append(results, AdminOrganizer{
				Id:        o.Id,
				Aliases:   o.Aliases,
				NumEvents: o.NumEvents,
			})
		}
	}

	c.JSON(http.StatusOK, results)
}

func (s *Server) MergeOrgs(c *gin.Context) {
	var req MergeOrgsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	if len(req.Ids) < 2 {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: "At least two organizer IDs are required to merge"})
		return
	}

	if err := s.Repo.MergeOrgs(req.Ids); err != nil {
		log.Printf("MergeOrgs error: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}
