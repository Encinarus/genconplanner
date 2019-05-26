package web

import (
	"github.com/Encinarus/genconplanner/internal/postgres"
	"sort"
)

type Context struct {
	Year        int
	DisplayName string
	Email       string
	Starred     *postgres.UserStarredEvents
}

func PartitionGroups(
	groups []*postgres.EventGroup,
	keyFunction func(*postgres.EventGroup) (string, string),
) ([]string, map[string][]string, map[string]map[string][]*postgres.EventGroup) {

	majorPartitions := make(map[string]map[string][]*postgres.EventGroup)
	majorKeys := make([]string, 0)
	minorKeys := make(map[string][]string)

	const soldOut = "Sold out"
	hasSoldOut := false

	for _, group := range groups {
		majorKey, minorKey := keyFunction(group)
		if group.TotalTickets == 0 {
			minorKey = majorKey
			majorKey = soldOut
			hasSoldOut = true
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
	if hasSoldOut && len(majorKeys) > 1 {
		index := sort.SearchStrings(majorKeys, soldOut)
		majorKeys = append(majorKeys[:index], majorKeys[index+1:]...)
		majorKeys = append(majorKeys, soldOut)
	}
	return majorKeys, minorKeys, majorPartitions
}
