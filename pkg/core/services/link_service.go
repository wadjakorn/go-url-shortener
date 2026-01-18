package services

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"time"

	"github.com/wadjakorntonsri/go-url-shortener/pkg/core/domain"
	"github.com/wadjakorntonsri/go-url-shortener/pkg/ports"
)

type LinkService struct {
	repo ports.LinkRepository
}

func NewLinkService(repo ports.LinkRepository) *LinkService {
	return &LinkService{repo: repo}
}

func (s *LinkService) Shorten(ctx context.Context, originalURL, title string, tags []string, customCode string) (*domain.Link, error) {
	if originalURL == "" {
		return nil, errors.New("original URL is required")
	}

	code := customCode
	if code == "" {
		var err error
		code, err = generateShortCode(6)
		if err != nil {
			return nil, err
		}
	} else {
		// Check if custom code exists
		existing, _ := s.repo.GetByShortCode(ctx, code)
		if existing != nil {
			return nil, errors.New("custom code already exists")
		}
	}

	link := &domain.Link{
		OriginalURL: originalURL,
		ShortCode:   code,
		Title:       title,
		Tags:        tags,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.repo.Create(ctx, link); err != nil {
		return nil, err
	}

	return link, nil
}

func (s *LinkService) GetOriginalURL(ctx context.Context, code string) (string, error) {
	link, err := s.repo.GetByShortCode(ctx, code)
	if err != nil {
		return "", err
	}
	if link == nil {
		return "", errors.New("link not found")
	}
	return link.OriginalURL, nil
}

func (s *LinkService) UpdateLink(ctx context.Context, id int64, originalURL, title string, tags []string) (*domain.Link, error) {
	link, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if link == nil {
		return nil, errors.New("link not found")
	}

	// Update fields if provided (naive partial update logic)
	if originalURL != "" {
		link.OriginalURL = originalURL
	}
	if title != "" {
		link.Title = title
	}
	if tags != nil {
		link.Tags = tags
	}
	link.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, link); err != nil {
		return nil, err
	}

	return link, nil
}

func (s *LinkService) DeleteLink(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *LinkService) ListLinks(ctx context.Context, page, limit int, search string, tag string) ([]domain.Link, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit

	filters := map[string]interface{}{
		"search": search,
		"tag":    tag,
	}

	links, err := s.repo.List(ctx, limit, offset, filters)
	if err != nil {
		return nil, 0, err
	}

	count, err := s.repo.Count(ctx, filters)
	if err != nil {
		return nil, 0, err
	}

	return links, count, nil
}

func (s *LinkService) RecordVisit(ctx context.Context, shortCode, referer, userAgent, ip string) error {
	link, err := s.repo.GetByShortCode(ctx, shortCode)
	if err != nil {
		return err
	}
	if link == nil {
		return errors.New("link not found")
	}

	// Simple privacy hash (in real app use salt)
	// For now just storing raw string or doing a dummy hash since verify isn't key
	ipHash := ip // In production: sha256.Sum256(ip + salt)

	visit := &domain.Visit{
		LinkID:    link.ID,
		Referer:   referer,
		UserAgent: userAgent,
		IPHash:    ipHash,
		CreatedAt: time.Now(),
	}

	// Async recording could be better for performance, but sync is safer for now.
	// To make it non-blocking for the redirect request, we could run in goroutine,
	// but context cancellation might kill it.
	// For MVP: Sync.
	return s.repo.RecordVisit(ctx, visit)
}

func (s *LinkService) GetLinkStats(ctx context.Context, id int64) (*domain.LinkStats, error) {
	return s.repo.GetLinkStats(ctx, id)
}

func (s *LinkService) GetDashboard(ctx context.Context, limit int, search, tag, domainFilter string) ([]domain.Link, int64, error) {
	if limit < 1 {
		limit = 10
	}
	filters := map[string]interface{}{
		"search": search,
		"tag":    tag,
		"domain": domainFilter,
	}
	return s.repo.GetDashboardStats(ctx, limit, filters)
}

func (s *LinkService) GetLinkByShortCode(ctx context.Context, code string) (*domain.Link, error) {
	link, err := s.repo.GetByShortCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if link == nil {
		return nil, errors.New("link not found")
	}
	return link, nil
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func generateShortCode(length int) (string, error) {
	b := make([]byte, length)
	for i := range b {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		b[i] = charset[num.Int64()]
	}
	return string(b), nil
}
