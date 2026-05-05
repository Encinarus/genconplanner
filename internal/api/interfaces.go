package api

import (
	"context"

	"github.com/Encinarus/genconplanner/internal/postgres"
)

// AuthService wraps authentication logic, typically Firebase.
type AuthService interface {
	VerifyIDToken(ctx context.Context, idToken string) (string, error)
}

// GameService wraps game metadata lookup logic.
type GameService interface {
	FindGame(name string) *postgres.Game
}

// ErrorResponse is the standard format for API error messages.
type ErrorResponse struct {
	Error string `json:"error"`
}
