package bgg

import (
	"encoding/xml"
	"os"
	"testing"
)

func TestParseGameItem(t *testing.T) {
	data, err := os.ReadFile("testing_data/catan.xml")
	if err != nil {
		t.Fatalf("Failed to read test data: %v", err)
	}

	var games Games
	err = xml.Unmarshal(data, &games)
	if err != nil {
		t.Fatalf("Failed to unmarshal XML: %v", err)
	}

	if len(games.Items) == 0 {
		t.Fatalf("Expected to find game items, got 0")
	}

	game := games.Items[0]

	if game.ID != 13 {
		t.Errorf("Expected Game ID 13, got %d", game.ID)
	}

	if game.MinPlayers.Value != 3 {
		t.Errorf("Expected MinPlayers 3, got %d", game.MinPlayers.Value)
	}

	if game.MaxPlayers.Value != 4 {
		t.Errorf("Expected MaxPlayers 4, got %d", game.MaxPlayers.Value)
	}

	if game.Description == "" {
		t.Errorf("Expected non-empty Description, got empty")
	}

	if len(game.Polls) == 0 {
		t.Errorf("Expected Polls, got 0")
	}

	foundPoll := false
	for _, poll := range game.Polls {
		if poll.Name == "suggested_numplayers" {
			foundPoll = true
			if len(poll.Results) == 0 {
				t.Errorf("Expected results in suggested_numplayers poll, got 0")
			}
			break
		}
	}
	if !foundPoll {
		t.Errorf("Expected to find suggested_numplayers poll")
	}

	if game.Statistics.Ratings.AverageWeight.Value == 0 {
		t.Errorf("Expected AverageWeight.Value to be > 0, got %v", game.Statistics.Ratings.AverageWeight.Value)
	}
	
	if game.Statistics.Ratings.NumWeights.Value == 0 {
		t.Errorf("Expected NumWeights.Value to be > 0, got %v", game.Statistics.Ratings.NumWeights.Value)
	}
}
