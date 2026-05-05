package api

import (
	"context"

	firebase "firebase.google.com/go"
	"github.com/Encinarus/genconplanner/internal/background"
	"github.com/Encinarus/genconplanner/internal/postgres"
)

type Server struct {
	Repo  EventRepository
	Auth  AuthService
	Games GameService
}

func NewServer(repo EventRepository, auth AuthService, games GameService) *Server {
	return &Server{
		Repo:  repo,
		Auth:  auth,
		Games: games,
	}
}

// FirebaseAuthWrapper implements AuthService using Firebase.
type FirebaseAuthWrapper struct {
	App *firebase.App
}

func (w *FirebaseAuthWrapper) VerifyIDToken(ctx context.Context, idToken string) (string, error) {
	client, err := w.App.Auth(ctx)
	if err != nil {
		return "", err
	}
	token, err := client.VerifyIDToken(ctx, idToken)
	if err != nil {
		return "", err
	}
	if token != nil {
		if email, ok := token.Claims["email"].(string); ok {
			return email, nil
		}
	}
	return "", nil
}

// GameCacheWrapper implements GameService using the background GameCache.
type GameCacheWrapper struct {
	Cache *background.GameCache
}

func (w *GameCacheWrapper) FindGame(name string) *postgres.Game {
	return w.Cache.FindGame(name)
}
