package domain

import "time"

// Collection represents a group of links (link-in-bio)
type Collection struct {
	ID          int64     `json:"id"`
	Slug        string    `json:"slug"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	LinkIDs     []int64   `json:"link_ids,omitempty"` // For convenience, though likely fetched separately
	Links       []Link    `json:"links,omitempty"`    // Populated when fetching full collection details
}

// CollectionLink represents the many-to-many relationship with ordering
type CollectionLink struct {
	CollectionID int64 `json:"collection_id"`
	LinkID       int64 `json:"link_id"`
	SortOrder    int   `json:"sort_order"`
}
