package perm

var roleRank = map[string]int{
	roleRen:     4,
	"admin":     3,
	"moderator": 2,
	"creator":   1,
}

func RoleRank(role string) int { return roleRank[role] }

func Rank(roles []string) int {
	best := 0
	for _, r := range roles {
		if v := roleRank[r]; v > best {
			best = v
		}
	}
	return best
}
