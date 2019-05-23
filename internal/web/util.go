package web

import (
	"github.com/Encinarus/genconplanner/internal/postgres"
	"sort"
	"strings"
)

type Context struct {
	Year        int
	DisplayName string
	Email       string
	Starred     *postgres.UserStarredEvents
}

func ParseDayQuery(rawDays string) []int {
	processedDays := make([]int, 0)
	splitDays := strings.Split(strings.ToLower(rawDays), ",")
	for _, day := range splitDays {
		switch day {
		case "sun":
			processedDays = append(processedDays, 0)
			break
		case "wed":
			processedDays = append(processedDays, 3)
			break
		case "thu":
			processedDays = append(processedDays, 4)
			break
		case "fri":
			processedDays = append(processedDays, 5)
			break
		case "sat":
			processedDays = append(processedDays, 6)
			break
		}
	}
	if len(processedDays) == 0 {
		return []int{0, 3, 4, 5, 6}
	} else {
		return processedDays
	}
}

func PartitionGroups(
	groups []*postgres.EventGroup,
	keyFunction func(*postgres.EventGroup) (string, string),
) ([]string, map[string][]string, map[string]map[string][]*postgres.EventGroup) {

	majorPartitions := make(map[string]map[string][]*postgres.EventGroup)
	majorKeys := make([]string, 0)
	minorKeys := make(map[string][]string)

	const soldOut = "Sold out"

	for _, group := range groups {
		majorKey, minorKey := keyFunction(group)
		if group.TotalTickets == 0 {
			majorKey = soldOut
			minorKey = majorKey
		}
		if _, found := majorPartitions[majorKey]; !found {
			majorPartitions[majorKey] = make(map[string][]*postgres.EventGroup)
			majorKeys = append(majorKeys, majorKey)
			minorKeys[majorKey] = make([]string, 0)
		}
		if _, found := majorPartitions[majorKey][minorKey]; !found {
			majorPartitions[majorKey][minorKey] = make([]*postgres.EventGroup, 0)
			// First time encountering this minor key, add to the list
			minorKeys[majorKey] = append(minorKeys[majorKey], minorKey)
		}
		majorPartitions[majorKey][minorKey] = append(majorPartitions[majorKey][minorKey], group)
	}
	sort.Strings(majorKeys)
	for k := range minorKeys {
		sort.Strings(minorKeys[k])
	}
	// Now that we've sorted, move sold out to the end
	index := sort.SearchStrings(majorKeys, soldOut)
	if index > 0 && len(majorKeys) > 1 {
		majorKeys = append(majorKeys[:index], majorKeys[index+1:]...)
		majorKeys = append(majorKeys, soldOut)
	}
	return majorKeys, minorKeys, majorPartitions
}
