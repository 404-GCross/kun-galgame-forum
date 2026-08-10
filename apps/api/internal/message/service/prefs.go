package service

import "strings"

const (
	KeyChat    = "chat"
	wikiPrefix = "wiki:"
)

var LocalNotificationTypes = []string{
	string(NotifyUpvoted), string(NotifyLiked), string(NotifyFavorite),
	string(NotifyReplied), string(NotifyCommented), string(NotifyMentioned),
	string(NotifySolution), string(NotifyPinReply), string(NotifyExpired),
	string(NotifyRequested), string(NotifyMerged), string(NotifyDeclined),
	"quiz-answered",
}

var WikiNotificationKeys = []string{
	"wiki:approved", "wiki:declined", "wiki:banned", "wiki:unbanned",
}

var allNotificationKeys = func() map[string]bool {
	m := make(map[string]bool)
	for _, k := range LocalNotificationTypes {
		m[k] = true
	}
	m[KeyChat] = true
	for _, k := range WikiNotificationKeys {
		m[k] = true
	}
	return m
}()

func SanitizeMutedKeys(keys []string) []string {
	seen := make(map[string]bool, len(keys))
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		if allNotificationKeys[k] && !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	return out
}

func SplitMuted(muted []string) (local []string, chatMuted bool) {
	for _, k := range muted {
		switch {
		case k == KeyChat:
			chatMuted = true
		case strings.HasPrefix(k, wikiPrefix):
		default:
			local = append(local, k)
		}
	}
	return local, chatMuted
}
