package services

import (
	"context"
	"errors"
	"time"

	"github.com/wadjakorntonsri/go-url-shortener/pkg/core/domain"
	"github.com/wadjakorntonsri/go-url-shortener/pkg/ports"
)

type CollectionService struct {
	repo ports.LinkRepository
}

func NewCollectionService(repo ports.LinkRepository) *CollectionService {
	return &CollectionService{repo: repo}
}

func (s *CollectionService) CreateCollection(ctx context.Context, title, slug, description string) (*domain.Collection, error) {
	if slug == "" {
		return nil, errors.New("slug is required")
	}

	// Check if slug exists
	existing, _ := s.repo.GetCollectionBySlug(ctx, slug)
	if existing != nil {
		return nil, errors.New("slug already exists")
	}

	collection := &domain.Collection{
		Title:       title,
		Slug:        slug,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.repo.CreateCollection(ctx, collection); err != nil {
		return nil, err
	}

	return collection, nil
}

func (s *CollectionService) GetCollection(ctx context.Context, id int64) (*domain.Collection, error) {
	collection, err := s.repo.GetCollection(ctx, id)
	if err != nil {
		return nil, err
	}
	if collection != nil {
		links, err := s.repo.GetCollectionLinks(ctx, id)
		if err != nil {
			return nil, err
		}
		collection.Links = links
	}
	return collection, nil
}

func (s *CollectionService) GetCollectionBySlug(ctx context.Context, slug string) (*domain.Collection, error) {
	collection, err := s.repo.GetCollectionBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if collection != nil {
		links, err := s.repo.GetCollectionLinks(ctx, collection.ID)
		if err != nil {
			return nil, err
		}
		collection.Links = links
	}
	return collection, nil
}

func (s *CollectionService) UpdateCollection(ctx context.Context, id int64, title, slug, description string) (*domain.Collection, error) {
	collection, err := s.repo.GetCollection(ctx, id)
	if err != nil {
		return nil, err
	}
	if collection == nil {
		return nil, errors.New("collection not found")
	}

	// Check slug uniqueness if changed
	if slug != collection.Slug {
		existing, _ := s.repo.GetCollectionBySlug(ctx, slug)
		if existing != nil {
			return nil, errors.New("slug already exists")
		}
	}

	collection.Title = title
	collection.Slug = slug
	collection.Description = description
	collection.UpdatedAt = time.Now()

	if err := s.repo.UpdateCollection(ctx, collection); err != nil {
		return nil, err
	}

	return collection, nil
}

func (s *CollectionService) DeleteCollection(ctx context.Context, id int64) error {
	return s.repo.DeleteCollection(ctx, id)
}

func (s *CollectionService) ListCollections(ctx context.Context, page, limit int, search string) ([]domain.Collection, int64, error) {
	offset := (page - 1) * limit
	filters := map[string]interface{}{}
	if search != "" {
		filters["search"] = search
	}

	collections, err := s.repo.ListCollections(ctx, limit, offset, filters)
	if err != nil {
		return nil, 0, err
	}

	// We don't have a count method for collections yet, returning 0 for total temporarily or implementing CountCollections if strictly needed.
	// For now, let's just return what we have.
	return collections, 0, nil
}

func (s *CollectionService) AddLink(ctx context.Context, collectionID, linkID int64) error {
	return s.repo.AddLinkToCollection(ctx, collectionID, linkID)
}

func (s *CollectionService) RemoveLink(ctx context.Context, collectionID, linkID int64) error {
	return s.repo.RemoveLinkFromCollection(ctx, collectionID, linkID)
}

func (s *CollectionService) ReorderLinks(ctx context.Context, collectionID int64, linkIDs []int64) error {
	for i, linkID := range linkIDs {
		// New order is index + 1
		if err := s.repo.UpdateLinkOrder(ctx, collectionID, linkID, i+1); err != nil {
			return err
		}
	}
	return nil
}
