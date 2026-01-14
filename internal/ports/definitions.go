package ports

import (
	"context"

	"github.com/wadjakorntonsri/go-url-shortener/internal/core/domain"
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
	GetLinkStats(ctx context.Context, linkID int64) (*domain.LinkStats, error)
	GetDashboardStats(ctx context.Context, limit int, filters map[string]interface{}) ([]domain.Link, int64, error)
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
	GetLinkStats(ctx context.Context, id int64) (*domain.LinkStats, error)
	GetDashboard(ctx context.Context, limit int, search, tag, domainFilter string) ([]domain.Link, int64, error)
}
