package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCategoryValidation(t *testing.T) {
	// Set Gin to test mode to keep logs clean
	gin.SetMode(gin.TestMode)

	// Setup a router with just the category route for testing validation
	// We pass nil for the DB since validation happens before DB access for these cases
	r := gin.New()
	apiGroup := r.Group("/api/v1")
	categoryRoutes(apiGroup, nil)

	tests := []struct {
		name         string
		year         string
		expectedCode int
	}{
		{
			name:         "Invalid year format",
			year:         "abc",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Year before 2020",
			year:         "2019",
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/category/"+tt.year, nil)
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("Expected status code %d, got %d", tt.expectedCode, w.Code)
			}
		})
	}
}
