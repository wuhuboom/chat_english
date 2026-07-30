package controller

import (
	"sort"

	"go-fly-muti/models"
)

type kefuOnlineCheck func(string) bool

// selectAvailableKefu keeps an explicitly requested or previously assigned
// online agent sticky, then falls back to the least-loaded online agent.
func selectAvailableKefu(users []models.User, preferred []string, online kefuOnlineCheck, activeLoads map[string]int) (models.User, bool, string) {
	usersByName := make(map[string]models.User, len(users))
	for _, user := range users {
		usersByName[user.Name] = user
	}

	seen := make(map[string]struct{}, len(preferred))
	for _, name := range preferred {
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		user, ok := usersByName[name]
		if ok && user.OnlineStatus == 1 && online(user.Name) {
			return user, true, "preferred"
		}
	}

	available := make([]models.User, 0, len(users))
	for _, user := range users {
		if user.OnlineStatus == 1 && online(user.Name) {
			available = append(available, user)
		}
	}
	if len(available) == 0 {
		return models.User{}, false, "offline"
	}

	sort.SliceStable(available, func(left, right int) bool {
		leftLoad := activeLoads[available[left].Name]
		rightLoad := activeLoads[available[right].Name]
		if leftLoad != rightLoad {
			return leftLoad < rightLoad
		}
		if available[left].RecNum != available[right].RecNum {
			return available[left].RecNum < available[right].RecNum
		}
		return available[left].Name < available[right].Name
	})
	return available[0], true, "least_loaded"
}

func fallbackKefu(users []models.User, preferred []string, ent models.User) models.User {
	usersByName := make(map[string]models.User, len(users))
	for _, user := range users {
		usersByName[user.Name] = user
	}
	for _, name := range preferred {
		if user, ok := usersByName[name]; ok {
			return user
		}
	}
	return ent
}
