package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/wadjakorntonsri/go-url-shortener/pkg/ports"
)

type CollectionHandler struct {
	service ports.CollectionService
}

func NewCollectionHandler(service ports.CollectionService) *CollectionHandler {
	return &CollectionHandler{service: service}
}

type createCollectionRequest struct {
	Title       string `json:"title"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

func (h *CollectionHandler) CreateCollection(w http.ResponseWriter, r *http.Request) {
	var req createCollectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	collection, err := h.service.CreateCollection(r.Context(), req.Title, req.Slug, req.Description)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(collection)
}

func (h *CollectionHandler) ListCollections(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 {
		limit = 10
	}
	search := r.URL.Query().Get("search")

	collections, total, err := h.service.ListCollections(r.Context(), page, limit, search)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"data":  collections,
		"total": total, // This might be 0 for now as per service impl
		"page":  page,
		"limit": limit,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *CollectionHandler) GetCollection(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	collection, err := h.service.GetCollection(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if collection == nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(collection)
}

func (h *CollectionHandler) UpdateCollection(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var req createCollectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	collection, err := h.service.UpdateCollection(r.Context(), id, req.Title, req.Slug, req.Description)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(collection)
}

func (h *CollectionHandler) DeleteCollection(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.service.DeleteCollection(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type addLinkRequest struct {
	LinkID int64 `json:"link_id"`
}

func (h *CollectionHandler) AddLink(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	collectionID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid Collection ID", http.StatusBadRequest)
		return
	}

	var req addLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.service.AddLink(r.Context(), collectionID, req.LinkID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *CollectionHandler) RemoveLink(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	collectionID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid Collection ID", http.StatusBadRequest)
		return
	}

	linkIDStr := r.PathValue("linkID")
	linkID, err := strconv.ParseInt(linkIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid Link ID", http.StatusBadRequest)
		return
	}

	if err := h.service.RemoveLink(r.Context(), collectionID, linkID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *CollectionHandler) GetPublicCollection(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		http.Error(w, "Slug required", http.StatusBadRequest)
		return
	}

	collection, err := h.service.GetCollectionBySlug(r.Context(), slug)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if collection == nil {
		http.NotFound(w, r)
		return
	}

	// TODO: Fetch links for the collection here if they are not already populated
	// For now, returning the collection object. Ideally we should expand links.

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(collection)
}
