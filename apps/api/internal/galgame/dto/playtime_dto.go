package dto

type PlaytimeMineItem struct {
	Galgame      GalgameListCard `json:"galgame"`
	Minutes      int             `json:"minutes"`
	Status       string          `json:"status"`
	LastPlayedAt string          `json:"last_played_at,omitempty"`
	UpdatedAt    string          `json:"updated_at"`
	Clients      int             `json:"clients"`
	// True when the largest report on this work came from an application other
	// than the forum — a desktop tracker the user authorised themselves. The
	// forum must never silently overwrite one of those.
	External bool `json:"external"`
}

type PlaytimeMinePage struct {
	Items         []PlaytimeMineItem `json:"items"`
	Total         int                `json:"total"`
	TotalMinutes  int                `json:"total_minutes"`
	FinishedWorks int                `json:"finished_works"`
	// Set when the user reports on more works than one sweep of the upstream
	// sync face returns; the page is then the oldest-changed slice, not all of it.
	Truncated bool `json:"truncated"`
}
