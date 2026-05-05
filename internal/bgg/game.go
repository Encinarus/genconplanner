package bgg

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

var ErrNoApiKey = errors.New("BGG API key not set")

// XML tags generated from https://www.onlinetool.io/xmltogo/
// Game can be a game, or expansion, see the Item.Type field.
type Game struct {
	Item struct {
		Type          string `xml:"type,attr"`
		ID            int64  `xml:"id,attr"`
		YearPublished struct {
			Text  string `xml:",chardata"`
			Value int64  `xml:"value,attr"`
		} `xml:"yearpublished"`
		Name []struct {
			Type  string `xml:"type,attr"`
			Value string `xml:"value,attr"`
		} `xml:"name"`
		Description string `xml:"description"`
		Link        []struct {
			Type  string `xml:"type,attr"`
			ID    int64  `xml:"id,attr"`
			Value string `xml:"value,attr"`
		} `xml:"link"`
		Statistics struct {
			Ratings struct {
				Text       string `xml:",chardata"`
				NumRatings struct {
					Text  string `xml:",chardata"`
					Value int64  `xml:"value,attr"`
				} `xml:"usersrated"`
				Average struct {
					Text  string  `xml:",chardata"`
					Value float64 `xml:"value,attr"`
				} `xml:"average"`
			} `xml:"ratings"`
		} `xml:"statistics"`
	} `xml:"item"`
}

type Family struct {
	Item struct {
		Type string `xml:"type,attr"`
		ID   int64  `xml:"id,attr"`
		Name struct {
			Value string `xml:"value,attr"`
		} `xml:"name"`
		Link []struct {
			Type  string `xml:"type,attr"`
			ID    int64  `xml:"id,attr"`
			Value string `xml:"value,attr"`
		} `xml:"link"`
	} `xml:"item"`
}

type BggApi struct {
	limiter *rate.Limiter
	apiKey  string
}

func NewBggApi(apiKey string) *BggApi {
	return &BggApi{
		limiter: rate.NewLimiter(rate.Every(5*time.Second), 1),
		apiKey:  apiKey,
	}
}

func (bgg *BggApi) get(ctx context.Context, url string, v interface{}) error {
	if bgg.apiKey == "" {
		log.Println("BGG API key not set, skipping request.")
		return ErrNoApiKey
	}

	err := bgg.limiter.Wait(ctx)
	if err != nil {
		return err
	}

	client := &http.Client{}

	for {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+bgg.apiKey)
		req.Header.Set("User-Agent", "GenConPlanner/1.0 (+https://github.com/Encinarus/genconplanner)")

		resp, err := client.Do(req)
		if err != nil {
			return err
		}

		if resp.StatusCode == http.StatusAccepted {
			resp.Body.Close()
			log.Printf("BGG API returned 202 (Accepted) for %s, waiting 5 seconds to retry...", url)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(5 * time.Second):
				continue
			}
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("Surprise status code: %v", resp.StatusCode)
		}

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error in processing body %v", err)
			return err
		}
		return xml.Unmarshal(bodyBytes, v)
	}
}

func (bgg *BggApi) GetGame(ctx context.Context, id int64) (*Game, error) {
	url := fmt.Sprintf("http://boardgamegeek.com/xmlapi2/thing?type=boardgame,boardgameexpansion&stats=1&id=%d", id)
	var game Game
	err := bgg.get(ctx, url, &game)
	if err != nil {
		return nil, err
	}
	if len(game.Item.Name) == 0 {
		return nil, errors.New("Not a board game")
	}
	return &game, nil
}

func (bgg *BggApi) GetFamily(ctx context.Context, id int64) (*Family, error) {
	url := fmt.Sprintf("http://boardgamegeek.com/xmlapi2/family?id=%d", id)
	var family Family
	err := bgg.get(ctx, url, &family)
	if err != nil {
		return nil, err
	}
	return &family, nil
}
