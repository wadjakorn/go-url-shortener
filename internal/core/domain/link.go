package domain

import "time"

// Link represents a shortened URL
type Link struct {
	ID          int64      `json:"id"`
	OriginalURL string     `json:"original_url"`
	ShortCode   string     `json:"short_code"`
	Title       string     `json:"title"`
	Tags        []string   `json:"tags"` // Handled as JSON text in SQLite
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
	Clicks      int64      `json:"clicks,omitempty"` // Aggregated count
}
