package domain

import "time"

// Visit represents a click on a short link
type Visit struct {
	ID        int64     `json:"id"`
	LinkID    int64     `json:"link_id"`
	Referer   string    `json:"referer"`
	UserAgent string    `json:"user_agent"`
	IPHash    string    `json:"ip_hash"` // Anonymized IP
	CreatedAt time.Time `json:"created_at"`
}

// Stats represents aggregated statistics for a link
type LinkStats struct {
	TotalClicks int64            `json:"total_clicks"`
	Referrers   map[string]int64 `json:"referrers"`    // count by domain
	DailyClicks []DailyClick     `json:"daily_clicks"` // timeline
}

type DailyClick struct {
	Date  string `json:"date"` // YYYY-MM-DD
	Count int64  `json:"count"`
}
