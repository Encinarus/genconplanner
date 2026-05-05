package api

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type Category struct {
	Name       string `json:"name"`
	Code       string `json:"code"`
	EventCount int    `json:"eventCount"`
	Year       int    `json:"year"`
}

func (s *Server) ListCategories(c *gin.Context) {
	year := 0
	var err error
	if len(strings.TrimSpace(c.Param("year"))) > 0 {
		year, err = strconv.Atoi(c.Param("year"))
		if err != nil {
			c.AbortWithStatusJSON(400, ErrorResponse{Error: "Invalid year"})
			return
		}
	}

	if year < 2020 {
		c.AbortWithStatusJSON(400, ErrorResponse{Error: "Year must be 2020 or later"})
		return
	}

	summary, err := s.Repo.LoadCategorySummary(year)

	if err != nil {
		c.AbortWithStatusJSON(500, ErrorResponse{Error: "Internal server error"})
		return
	}

	results := make([]Category, 0)
	for i := range summary {
		results = append(results, Category{
			Name:       summary[i].Name,
			Code:       summary[i].Code,
			EventCount: summary[i].Count,
			Year:       year,
		})
	}

	c.JSON(200, results)
}
