package web

import (
	"github.com/Encinarus/genconplanner/internal/postgres"
	"sort"
)

type Context struct {
	Year        int
	DisplayName string
	Email       string
}

func PartitionGroups(
	groups []*postgres.EventGroup,
	keyFunction func(*postgres.EventGroup) string,
) ([]string, map[string][]*postgres.EventGroup) {

	partitions := make(map[string][]*postgres.EventGroup)
	keys := make([]string, 0)

	for _, group := range groups {
		key := keyFunction(group)
		partition, ok := partitions[key]
		if !ok {
			partition = make([]*postgres.EventGroup, 0)
			keys = append(keys, key)
		}
		partitions[key] = append(partition, group)
	}
	sort.Strings(keys)
	return keys, partitions
}
