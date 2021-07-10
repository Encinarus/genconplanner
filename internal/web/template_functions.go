package web

import (
	"fmt"
	"github.com/Encinarus/genconplanner/internal/background"
	"html/template"
	"regexp"
	"strings"
)

var textStrippingRegex, _ = regexp.Compile("[^a-zA-Z0-9]+")

func textToId(text string) string {
	return textStrippingRegex.ReplaceAllString(strings.ToLower(text), "")
}

func GetTemplateFunctions(cache *background.GameCache) template.FuncMap {
	return template.FuncMap{
		"toId":          textToId,
		"dict":          dict,
		"bggPage":       func(gameName string) string { return bggPage(gameName, cache) },
		"bggRating":     func(gameName string) string { return bggRating(gameName, cache) },
		"bggNumRatings": func(gameName string) string { return bggNumRatings(gameName, cache) },
	}
}

func dict(v ...interface{}) map[string]interface{} {
	dict := map[string]interface{}{}
	lenv := len(v)
	for i := 0; i < lenv; i += 2 {
		key := fmt.Sprintf("%s", v[i])
		if i+1 >= lenv {
			dict[key] = ""
			continue
		}
		dict[key] = v[i+1]
	}
	return dict
}

func bggPage(gameName string, cache *background.GameCache) string {
	bggGame := cache.FindGame(gameName)
	if bggGame == nil {
		return ""
	}

	return fmt.Sprintf("https://boardgamegeek.com/boardgame/%d", bggGame.BggId)
}

func bggRating(gameName string, cache *background.GameCache) string {
	bggGame := cache.FindGame(gameName)
	if bggGame == nil || bggGame.AvgRatings < 0.1 {
		return ""
	}

	return fmt.Sprintf("%2.1f", bggGame.AvgRatings)
}

func bggNumRatings(gameName string, cache *background.GameCache) string {
	bggGame := cache.FindGame(gameName)
	if bggGame == nil || bggGame.NumRatings == 0 {
		return ""
	}

	return fmt.Sprintf("%d", bggGame.NumRatings)
}
