package userclient

import "context"

func CollectIDs[T any](rows []T, idOf func(T) int) []int {
	if len(rows) == 0 {
		return nil
	}
	seen := make(map[int]struct{}, len(rows))
	ids := make([]int, 0, len(rows))
	for _, r := range rows {
		id := idOf(r)
		if id <= 0 {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

func DerefID(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func (c *Client) Hydrate(ctx context.Context, ids []int) map[int]User {
	out := make(map[int]User, len(ids))
	if len(ids) == 0 {
		return out
	}
	users, _ := c.Users(ctx, ids)
	for _, id := range ids {
		if u, ok := users[id]; ok {
			out[id] = u
		} else {
			out[id] = Placeholder(id)
		}
	}
	return out
}

func IsRenderable(u User) bool {
	return u.Status == 0
}
