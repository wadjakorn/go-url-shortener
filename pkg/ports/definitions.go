package ports

import (
	"context"

	"github.com/wadjakorntonsri/go-url-shortener/pkg/core/domain"
)

// LinkRepository defines storage operations for links
type LinkRepository interface {
	Create(ctx context.Context, link *domain.Link) error
	GetByShortCode(ctx context.Context, code string) (*domain.Link, error)
	GetByID(ctx context.Context, id int64) (*domain.Link, error)
	Update(ctx context.Context, link *domain.Link) error
	Delete(ctx context.Context, id int64) error // Soft delete
	List(ctx context.Context, limit, offset int, filters map[string]interface{}) ([]domain.Link, error)
	Count(ctx context.Context, filters map[string]interface{}) (int64, error)
	Dump(ctx context.Context) ([]domain.Link, error) // For migration

	// Stats
	RecordVisit(ctx context.Context, visit *domain.Visit) error
	GetLinkStats(ctx context.Context, linkID int64, filters map[string]interface{}) (*domain.LinkStats, error)
	GetDashboardStats(ctx context.Context, limit int, filters map[string]interface{}) ([]domain.Link, int64, error)

	// Collections
	CreateCollection(ctx context.Context, collection *domain.Collection) error
	GetCollection(ctx context.Context, id int64) (*domain.Collection, error)
	GetCollectionBySlug(ctx context.Context, slug string) (*domain.Collection, error)
	UpdateCollection(ctx context.Context, collection *domain.Collection) error
	DeleteCollection(ctx context.Context, id int64) error
	ListCollections(ctx context.Context, limit, offset int, filters map[string]interface{}) ([]domain.Collection, error)
	AddLinkToCollection(ctx context.Context, collectionID, linkID int64) error
	RemoveLinkFromCollection(ctx context.Context, collectionID, linkID int64) error
	UpdateLinkOrder(ctx context.Context, collectionID, linkID int64, newOrder int) error
	GetCollectionLinks(ctx context.Context, collectionID int64) ([]domain.Link, error)
} // LinkRepository ends here

// CollectionService defines business logic for collections
type CollectionService interface {
	CreateCollection(ctx context.Context, title, slug, description string) (*domain.Collection, error)
	GetCollection(ctx context.Context, id int64) (*domain.Collection, error)
	GetCollectionBySlug(ctx context.Context, slug string) (*domain.Collection, error)
	UpdateCollection(ctx context.Context, id int64, title, slug, description string) (*domain.Collection, error)
	DeleteCollection(ctx context.Context, id int64) error
	ListCollections(ctx context.Context, page, limit int, search string) ([]domain.Collection, int64, error)
	AddLink(ctx context.Context, collectionID, linkID int64) error
	RemoveLink(ctx context.Context, collectionID, linkID int64) error
	ReorderLinks(ctx context.Context, collectionID int64, linkIDs []int64) error
}

// LinkService defines the business logic operations
type LinkService interface {
	Shorten(ctx context.Context, originalURL, title string, tags []string, customCode string) (*domain.Link, error)
	GetOriginalURL(ctx context.Context, code string) (string, error)
	UpdateLink(ctx context.Context, id int64, originalURL, title string, tags []string) (*domain.Link, error)
	DeleteLink(ctx context.Context, id int64) error
	ListLinks(ctx context.Context, page, limit int, search string, tag string) ([]domain.Link, int64, error)

	// Stats
	RecordVisit(ctx context.Context, shortCode, referer, userAgent, ip string) error
	GetLinkStats(ctx context.Context, id int64, filters map[string]interface{}) (*domain.LinkStats, error)
	GetDashboard(ctx context.Context, limit int, search, tag, domainFilter string) ([]domain.Link, int64, error)
	GetLinkByShortCode(ctx context.Context, code string) (*domain.Link, error)
}
