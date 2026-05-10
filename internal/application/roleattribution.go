package application

import "math/rand"

func (a *App) pickRole() Role {
	availableRoleList := a.listAvailableRoles()
	n := len(availableRoleList)
	choice := rand.Intn(n)
	role := availableRoleList[choice]
	return role
}

func (a *App) listAvailableRoles() []Role {
	n := len(a.state.Players)

	availableRoleCount := map[Role]int{
		RoleWitch:    1,
		RoleWolf:     n / 3,
		RoleVillager: n - (n / 3) - 1,
	}

	for _, player := range a.state.Players {
		availableRoleCount[player.Role] -= 1
	}

	var availableRolesList []Role
	for role, count := range availableRoleCount {
		for i := 0; i < count; i++ {
			{
				availableRolesList = append(availableRolesList, role)
			}
		}
	}

	return availableRolesList
}

func (a *App) applyAttribution(id string, role Role) {
	p := a.state.Players[id]
	p.Role = role
	a.state.Players[id] = p
}

func (a *App) checkEveryoneHasRole() bool {
	for _, player := range a.state.Players {
		if player.Role == RoleUnknown {
			return false
		}
	}
	return true
}
