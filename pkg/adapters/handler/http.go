package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/wadjakorntonsri/go-url-shortener/pkg/ports"
)

type HTTPHandler struct {
	service ports.LinkService
}

func NewHTTPHandler(service ports.LinkService) *HTTPHandler {
	return &HTTPHandler{service: service}
}

// CreateLinkRequest payload
type CreateLinkRequest struct {
	OriginalURL string   `json:"original_url"`
	Title       string   `json:"title"`
	Tags        []string `json:"tags"`
	CustomCode  string   `json:"custom_code,omitempty"`
}

// UpdateLinkRequest payload
type UpdateLinkRequest struct {
	OriginalURL string   `json:"original_url,omitempty"`
	Title       string   `json:"title,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// Create Link
func (h *HTTPHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	link, err := h.service.Shorten(r.Context(), req.OriginalURL, req.Title, req.Tags, req.CustomCode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(link)
}

// Redirect to original URL
func (h *HTTPHandler) Redirect(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("short_code")
	if code == "" {
		// Fallback for older Go versions if not using new mux, but 1.22+ supports PathValue
		// Assuming using http.NewServeMux with pattern "/{short_code}"
		http.Error(w, "Short code missing", http.StatusBadRequest)
		return
	}

	originalURL, err := h.service.GetOriginalURL(r.Context(), code)
	if err != nil {
		http.Error(w, "Link not found", http.StatusNotFound)
		return
	}

	// Async track visit (only if query param "no_stat" is not set)
	if r.URL.Query().Get("no_stat") == "" {
		go func() {
			// Use background context as request context will be cancelled
			// Getting IP/UserAgent
			referer := r.Header.Get("Referer")
			userAgent := r.UserAgent()
			ip := r.RemoteAddr // naive
			_ = h.service.RecordVisit(context.Background(), code, referer, userAgent, ip)
		}()
	}

	http.Redirect(w, r, originalURL, http.StatusFound)
}

// Get Public Link (without redirect, for metadata resolution)
func (h *HTTPHandler) GetPublicByShortCode(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("short_code")
	if code == "" {
		http.Error(w, "Short code missing", http.StatusBadRequest)
		return
	}

	link, err := h.service.GetLinkByShortCode(r.Context(), code)
	if err != nil {
		http.Error(w, "Link not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(link)
}

// Track visit manually
func (h *HTTPHandler) Track(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("short_code")
	if code == "" {
		http.Error(w, "Short code missing", http.StatusBadRequest)
		return
	}

	// Async or Sync tracking
	referer := r.Header.Get("Referer")
	userAgent := r.UserAgent()
	ip := r.RemoteAddr

	if err := h.service.RecordVisit(r.Context(), code, referer, userAgent, ip); err != nil {
		// Just log error or ignore, don't break flow?
		// For now return error to client so they know
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Get Stats for a Link
func (h *HTTPHandler) Stats(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	stats, err := h.service.GetLinkStats(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(stats)
}

// Get Dashboard
func (h *HTTPHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	search := r.URL.Query().Get("search")
	tag := r.URL.Query().Get("tag")
	domainFilter := r.URL.Query().Get("domain")

	links, total, err := h.service.GetDashboard(r.Context(), limit, search, tag, domainFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"top_links":           links,
		"total_system_clicks": total,
	}
	json.NewEncoder(w).Encode(resp)
}

// List Links
func (h *HTTPHandler) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	search := r.URL.Query().Get("search")
	tag := r.URL.Query().Get("tag")

	links, count, err := h.service.ListLinks(r.Context(), page, limit, search, tag)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"data":  links,
		"total": count,
		"page":  page,
		"limit": limit,
	}
	json.NewEncoder(w).Encode(resp)
}

// Update Link
func (h *HTTPHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var req UpdateLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	link, err := h.service.UpdateLink(r.Context(), id, req.OriginalURL, req.Title, req.Tags)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(link)
}

// Delete Link
func (h *HTTPHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.service.DeleteLink(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
