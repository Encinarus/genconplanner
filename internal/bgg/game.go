package bgg

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"

	"log"
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

var ErrNoApiKey = errors.New("BGG API key not set")

type BGGClient interface {
	GetGames(ctx context.Context, ids []int64) ([]*GameItem, error)
	GetFamilies(ctx context.Context, ids []int64) ([]*FamilyItem, error)
}


// XML tags generated from https://www.onlinetool.io/xmltogo/
// Game can be a game, or expansion, see the Item.Type field.
type Games struct {
	Items []GameItem `xml:"item"`
}

type GameItem struct {
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
	MinPlayers struct {
		Value int64 `xml:"value,attr"`
	} `xml:"minplayers"`
	MaxPlayers struct {
		Value int64 `xml:"value,attr"`
	} `xml:"maxplayers"`
	Polls []struct {
		Name       string `xml:"name,attr"`
		Title      string `xml:"title,attr"`
		TotalVotes int64  `xml:"totalvotes,attr"`
		Results    []struct {
			NumPlayers string `xml:"numplayers,attr"`
			Result     []struct {
				Value    string `xml:"value,attr"`
				NumVotes int64  `xml:"numvotes,attr"`
			} `xml:"result"`
		} `xml:"results"`
	} `xml:"poll"`
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
			NumWeights struct {
				Text  string `xml:",chardata"`
				Value int64  `xml:"value,attr"`
			} `xml:"numweights"`
			AverageWeight struct {
				Text  string  `xml:",chardata"`
				Value float64 `xml:"value,attr"`
			} `xml:"averageweight"`
		} `xml:"ratings"`
	} `xml:"statistics"`
}

type Families struct {
	Items []FamilyItem `xml:"item"`
}

type FamilyItem struct {
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
}

type BggApi struct {
	limiter *rate.Limiter
	apiKey  string
}

func NewBggApi(apiKey string) *BggApi {
	return &BggApi{
		limiter: rate.NewLimiter(rate.Every(10*time.Second), 1),
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
			log.Printf("BGG API returned 202 (Accepted) for %s, waiting 4 seconds to retry...", url)
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

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error in processing body %v", err)
			return err
		}
		return xml.Unmarshal(bodyBytes, v)
	}
}

func (bgg *BggApi) GetGames(ctx context.Context, ids []int64) ([]*GameItem, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	idStr := ""
	for i, id := range ids {
		if i > 0 {
			idStr += ","
		}
		idStr += fmt.Sprintf("%d", id)
	}

	url := fmt.Sprintf("http://boardgamegeek.com/xmlapi2/thing?type=boardgame,boardgameexpansion&stats=1&id=%s", idStr)
	var games Games
	err := bgg.get(ctx, url, &games)
	if err != nil {
		return nil, err
	}

	res := make([]*GameItem, 0)
	for i := range games.Items {
		if len(games.Items[i].Name) == 0 {
			continue
		}
		res = append(res, &games.Items[i])
	}
	return res, nil
}

func (bgg *BggApi) GetFamilies(ctx context.Context, ids []int64) ([]*FamilyItem, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	idStr := ""
	for i, id := range ids {
		if i > 0 {
			idStr += ","
		}
		idStr += fmt.Sprintf("%d", id)
	}

	url := fmt.Sprintf("http://boardgamegeek.com/xmlapi2/family?id=%s", idStr)
	var families Families
	err := bgg.get(ctx, url, &families)
	if err != nil {
		return nil, err
	}

	res := make([]*FamilyItem, 0, len(families.Items))
	for i := range families.Items {
		res = append(res, &families.Items[i])
	}
	return res, nil
}
