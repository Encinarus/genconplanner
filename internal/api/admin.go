package api

import (
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"

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

type EventSample struct {
	Year   int      `json:"year"`
	Titles []string `json:"titles"`
}

type MergeSuggestion struct {
	Id           int64         `json:"id"`
	Aliases      []string      `json:"aliases"`
	NumEvents    int64         `json:"numEvents"`
	Reasons      []string      `json:"reasons"`
	EventSamples []EventSample `json:"eventSamples"`
}

type OrganizerWithSuggestions struct {
	Id           int64             `json:"id"`
	Aliases      []string          `json:"aliases"`
	NumEvents    int64             `json:"numEvents"`
	EventSamples []EventSample     `json:"eventSamples"`
	Suggestions  []MergeSuggestion `json:"suggestions"`
}

func levenshteinDistance(s, t string) int {
	d := make([][]int, len(s)+1)
	for i := range d {
		d[i] = make([]int, len(t)+1)
		d[i][0] = i
	}
	for j := range d[0] {
		d[0][j] = j
	}
	for i := 1; i <= len(s); i++ {
		for j := 1; j <= len(t); j++ {
			if s[i-1] == t[j-1] {
				d[i][j] = d[i-1][j-1]
			} else {
				minVal := d[i-1][j]
				if d[i][j-1] < minVal {
					minVal = d[i][j-1]
				}
				if d[i-1][j-1] < minVal {
					minVal = d[i-1][j-1]
				}
				d[i][j] = minVal + 1
			}
		}
	}
	return d[len(s)][len(t)]
}

func normalizeString(s string) string {
	s = strings.ToLower(s)
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, "&", "and")
	return s
}

func splitGMs(gmNames string) []string {
	var gms []string
	parts := strings.FieldsFunc(gmNames, func(r rune) bool {
		return r == ',' || r == ';'
	})
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			gms = append(gms, p)
		}
	}
	return gms
}

func isGenericGM(gm string) bool {
	l := strings.ToLower(gm)
	return l == "game master" || l == "staff" || l == "anonymous" || l == "null" || l == "tbd" || l == "none" || len(gm) < 3
}

func isGenericEmail(email string) bool {
	l := strings.ToLower(email)
	return l == "" || strings.Contains(l, "gencon.com") || strings.Contains(l, "example.com")
}

func isGenericWebsite(web string) bool {
	l := strings.ToLower(web)
	return l == "" || strings.Contains(l, "gencon.com") || strings.Contains(l, "example.com") || strings.Contains(l, "facebook.com") || strings.Contains(l, "twitter.com")
}

func getEventSamples(titleMap map[int]map[string]bool) []EventSample {
	var samples []EventSample
	var years []int
	for y := range titleMap {
		years = append(years, y)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(years)))

	if len(years) > 3 {
		years = years[:3]
	}

	for _, y := range years {
		var titles []string
		for t := range titleMap[y] {
			titles = append(titles, t)
			if len(titles) >= 3 {
				break
			}
		}
		sort.Strings(titles)
		samples = append(samples, EventSample{
			Year:   y,
			Titles: titles,
		})
	}
	return samples
}

func (s *Server) GetMergeSuggestions(c *gin.Context) {
	orgs, err := s.Repo.LoadAllOrgs()
	if err != nil {
		log.Printf("GetMergeSuggestions error: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	metadata, err := s.Repo.LoadEventOrgMetadata()
	if err != nil {
		log.Printf("GetMergeSuggestions error: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	aliasToId := make(map[string]int64)
	for _, o := range orgs {
		for _, a := range o.Aliases {
			aliasToId[strings.ToLower(a)] = o.Id
		}
	}

	type orgData struct {
		id           int64
		aliases      []string
		numEvents    int64
		gms          map[string]bool
		emails       map[string]bool
		websites     map[string]bool
		titles       map[string]bool
		yearlyTitles map[int]map[string]bool
	}

	dataMap := make(map[int64]*orgData)
	for _, o := range orgs {
		dataMap[o.Id] = &orgData{
			id:           o.Id,
			aliases:      o.Aliases,
			numEvents:    o.NumEvents,
			gms:          make(map[string]bool),
			emails:       make(map[string]bool),
			websites:     make(map[string]bool),
			titles:       make(map[string]bool),
			yearlyTitles: make(map[int]map[string]bool),
		}
	}

	for _, m := range metadata {
		normOrg := strings.ToLower(m.OrgGroup)
		id, ok := aliasToId[normOrg]
		if !ok {
			continue
		}
		d := dataMap[id]
		if m.GmNames != "" {
			for _, gm := range splitGMs(m.GmNames) {
				if !isGenericGM(gm) {
					d.gms[strings.ToLower(gm)] = true
				}
			}
		}
		if m.Email != "" && !isGenericEmail(m.Email) {
			d.emails[strings.ToLower(m.Email)] = true
		}
		if m.Website != "" && !isGenericWebsite(m.Website) {
			d.websites[strings.ToLower(m.Website)] = true
		}
		if m.Title != "" {
			d.titles[strings.ToLower(m.Title)] = true
			if d.yearlyTitles[m.Year] == nil {
				d.yearlyTitles[m.Year] = make(map[string]bool)
			}
			d.yearlyTitles[m.Year][m.Title] = true
		}
	}

	var matchedPairs []OrganizerWithSuggestions
	orgIds := make([]int64, 0, len(orgs))
	for id := range dataMap {
		orgIds = append(orgIds, id)
	}
	sort.Slice(orgIds, func(i, j int) bool { return orgIds[i] < orgIds[j] })

	getReasons := func(a, b *orgData) []string {
		var reasons []string

		for _, aliasA := range a.aliases {
			for _, aliasB := range b.aliases {
				nA := normalizeString(aliasA)
				nB := normalizeString(aliasB)
				if nA == nB {
					reasons = append(reasons, "Identical name: '"+aliasA+"'")
					break
				}
				dist := levenshteinDistance(nA, nB)
				if dist <= 2 && len(nA) >= 5 && len(nB) >= 5 {
					reasons = append(reasons, fmt.Sprintf("Similar name: '%s' / '%s' (edit distance %d)", aliasA, aliasB, dist))
					break
				}
				if len(nA) >= 6 && len(nB) >= 6 && (strings.HasPrefix(nA, nB) || strings.HasPrefix(nB, nA) || strings.HasSuffix(nA, nB) || strings.HasSuffix(nB, nA)) {
					reasons = append(reasons, "Name prefix/suffix match: '"+aliasA+"' / '"+aliasB+"'")
					break
				}
			}
		}

		var sharedGMs []string
		for gm := range a.gms {
			if b.gms[gm] {
				sharedGMs = append(sharedGMs, gm)
			}
		}
		if len(sharedGMs) > 0 {
			sort.Strings(sharedGMs)
			reasons = append(reasons, "Shared GM(s): "+strings.Join(sharedGMs, ", "))
		}

		var sharedEmails []string
		for email := range a.emails {
			if b.emails[email] {
				sharedEmails = append(sharedEmails, email)
			}
		}
		if len(sharedEmails) > 0 {
			sort.Strings(sharedEmails)
			reasons = append(reasons, "Shared contact email: "+strings.Join(sharedEmails, ", "))
		}

		var sharedWebsites []string
		for web := range a.websites {
			if b.websites[web] {
				sharedWebsites = append(sharedWebsites, web)
			}
		}
		if len(sharedWebsites) > 0 {
			sort.Strings(sharedWebsites)
			reasons = append(reasons, "Shared website: "+strings.Join(sharedWebsites, ", "))
		}

		var sharedTitles []string
		for title := range a.titles {
			if len(title) >= 10 && b.titles[title] {
				sharedTitles = append(sharedTitles, title)
			}
		}
		if len(sharedTitles) > 0 {
			sort.Strings(sharedTitles)
			reasons = append(reasons, fmt.Sprintf("Shared event title(s): %d matching title(s)", len(sharedTitles)))
		}

		return reasons
	}

	suggestionsMap := make(map[int64][]MergeSuggestion)

	for i := 0; i < len(orgIds); i++ {
		for j := i + 1; j < len(orgIds); j++ {
			idA := orgIds[i]
			idB := orgIds[j]
			a := dataMap[idA]
			b := dataMap[idB]

			reasons := getReasons(a, b)
			if len(reasons) > 0 {
				suggestionsMap[idA] = append(suggestionsMap[idA], MergeSuggestion{
					Id:           idB,
					Aliases:      b.aliases,
					NumEvents:    b.numEvents,
					Reasons:      reasons,
					EventSamples: getEventSamples(b.yearlyTitles),
				})
				suggestionsMap[idB] = append(suggestionsMap[idB], MergeSuggestion{
					Id:           idA,
					Aliases:      a.aliases,
					NumEvents:    a.numEvents,
					Reasons:      reasons,
					EventSamples: getEventSamples(a.yearlyTitles),
				})
			}
		}
	}

	for _, o := range orgs {
		suggs := suggestionsMap[o.Id]
		if len(suggs) > 0 {
			sort.Slice(suggs, func(i, j int) bool {
				return len(suggs[i].Reasons) > len(suggs[j].Reasons)
			})
			matchedPairs = append(matchedPairs, OrganizerWithSuggestions{
				Id:           o.Id,
				Aliases:      o.Aliases,
				NumEvents:    o.NumEvents,
				EventSamples: getEventSamples(dataMap[o.Id].yearlyTitles),
				Suggestions:  suggs,
			})
		}
	}

	sort.Slice(matchedPairs, func(i, j int) bool {
		if len(matchedPairs[i].Suggestions) != len(matchedPairs[j].Suggestions) {
			return len(matchedPairs[i].Suggestions) > len(matchedPairs[j].Suggestions)
		}
		return matchedPairs[i].NumEvents > matchedPairs[j].NumEvents
	})

	c.JSON(http.StatusOK, matchedPairs)
}
