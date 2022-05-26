package bgg

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"golang.org/x/time/rate"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

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
}

func NewBggApi() *BggApi {
	return &BggApi{limiter: rate.NewLimiter(rate.Every(5*time.Second), 1)}
}

func (bgg *BggApi) get(ctx context.Context, url string, v interface{}) error {
	// log.Println(url)

	err := bgg.limiter.Wait(ctx)
	if err != nil {
		return err
	}

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("Surprise status code: %v", resp.StatusCode))
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error in procesing body %v", err)
		return err
	}
	return xml.Unmarshal(bodyBytes, v)
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
