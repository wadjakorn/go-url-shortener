package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/wadjakorntonsri/go-url-shortener/internal/core/domain"
	"github.com/wadjakorntonsri/go-url-shortener/internal/ports"
	_ "modernc.org/sqlite"
)

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(dbURL string) (*SQLiteRepository, error) {
	db, err := sql.Open("sqlite", dbURL)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	if err := migrate(db); err != nil {
		return nil, err
	}

	return &SQLiteRepository{db: db}, nil
}

func migrate(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS links (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		original_url TEXT NOT NULL,
		short_code TEXT NOT NULL UNIQUE,
		title TEXT,
		tags JSON,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		deleted_at DATETIME
	);
	CREATE INDEX IF NOT EXISTS idx_links_short_code ON links(short_code);
	
	CREATE TABLE IF NOT EXISTS visits (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		link_id INTEGER NOT NULL,
		referer TEXT,
		user_agent TEXT,
		ip_hash TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(link_id) REFERENCES links(id)
	);
	CREATE INDEX IF NOT EXISTS idx_visits_link_id ON visits(link_id);
	`
	_, err := db.Exec(query)
	return err
}

func (r *SQLiteRepository) Create(ctx context.Context, link *domain.Link) error {
	query := `INSERT INTO links (original_url, short_code, title, tags, created_at, updated_at) 
			  VALUES (?, ?, ?, ?, ?, ?) RETURNING id`

	tagsJSON, err := json.Marshal(link.Tags)
	if err != nil {
		return err
	}

	// SQLite returns ID via LastInsertId usually, but modernc supports returning?
	// Actually pure sqlite3 doesn't support RETURNING in older versions, but modern ones do.
	// To be safe/standard in sql driver:

	res, err := r.db.ExecContext(ctx, query, link.OriginalURL, link.ShortCode, link.Title, tagsJSON, link.CreatedAt, link.UpdatedAt)
	if err != nil {
		// In a real app check for constraints here
		return err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	link.ID = id
	return nil
}

func (r *SQLiteRepository) GetByShortCode(ctx context.Context, code string) (*domain.Link, error) {
	query := `SELECT id, original_url, short_code, title, tags, created_at, updated_at, deleted_at 
			  FROM links WHERE short_code = ? AND deleted_at IS NULL`

	var link domain.Link
	var tagsJSON []byte
	var deletedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, code).Scan(
		&link.ID, &link.OriginalURL, &link.ShortCode, &link.Title, &tagsJSON,
		&link.CreatedAt, &link.UpdatedAt, &deletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if deletedAt.Valid {
		link.DeletedAt = &deletedAt.Time
	}

	_ = json.Unmarshal(tagsJSON, &link.Tags)
	return &link, nil
}

func (r *SQLiteRepository) GetByID(ctx context.Context, id int64) (*domain.Link, error) {
	query := `SELECT id, original_url, short_code, title, tags, created_at, updated_at, deleted_at 
			  FROM links WHERE id = ? AND deleted_at IS NULL`

	var link domain.Link
	var tagsJSON []byte
	var deletedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&link.ID, &link.OriginalURL, &link.ShortCode, &link.Title, &tagsJSON,
		&link.CreatedAt, &link.UpdatedAt, &deletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if deletedAt.Valid {
		link.DeletedAt = &deletedAt.Time
	}

	_ = json.Unmarshal(tagsJSON, &link.Tags)
	return &link, nil
}

func (r *SQLiteRepository) Update(ctx context.Context, link *domain.Link) error {
	query := `UPDATE links SET original_url = ?, title = ?, tags = ?, updated_at = ? WHERE id = ?`

	tagsJSON, err := json.Marshal(link.Tags)
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, query, link.OriginalURL, link.Title, tagsJSON, link.UpdatedAt, link.ID)
	return err
}

func (r *SQLiteRepository) Delete(ctx context.Context, id int64) error {
	query := `UPDATE links SET deleted_at = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	return err
}

func (r *SQLiteRepository) List(ctx context.Context, limit, offset int, filters map[string]interface{}) ([]domain.Link, error) {
	query := `SELECT id, original_url, short_code, title, tags, created_at, updated_at 
			  FROM links WHERE deleted_at IS NULL`
	args := []interface{}{}

	if search, ok := filters["search"].(string); ok && search != "" {
		query += " AND (title LIKE ? OR original_url LIKE ?)"
		args = append(args, "%"+search+"%", "%"+search+"%")
	}

	// Tag filtering in SQLite JSON is strict, using LIKE for simplicity on raw string or proper json_each if available.
	// modernc sqlite supports json.
	if tag, ok := filters["tag"].(string); ok && tag != "" {
		// This is a naive check. Ideally use: EXISTS (SELECT 1 FROM json_each(links.tags) WHERE value = ?)
		query += " AND EXISTS (SELECT 1 FROM json_each(links.tags) WHERE value = ?)"
		args = append(args, tag)
	}

	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []domain.Link
	for rows.Next() {
		var l domain.Link
		var tagsJSON []byte
		if err := rows.Scan(&l.ID, &l.OriginalURL, &l.ShortCode, &l.Title, &tagsJSON, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(tagsJSON, &l.Tags)
		links = append(links, l)
	}

	return links, nil
}

func (r *SQLiteRepository) Count(ctx context.Context, filters map[string]interface{}) (int64, error) {
	query := `SELECT COUNT(*) FROM links WHERE deleted_at IS NULL`
	args := []interface{}{}

	if search, ok := filters["search"].(string); ok && search != "" {
		query += " AND (title LIKE ? OR original_url LIKE ?)"
		args = append(args, "%"+search+"%", "%"+search+"%")
	}
	if tag, ok := filters["tag"].(string); ok && tag != "" {
		query += " AND EXISTS (SELECT 1 FROM json_each(links.tags) WHERE value = ?)"
		args = append(args, tag)
	}

	var count int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}

func (r *SQLiteRepository) Dump(ctx context.Context) ([]domain.Link, error) {
	query := `SELECT id, original_url, short_code, title, tags, created_at, updated_at, deleted_at FROM links`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []domain.Link
	for rows.Next() {
		var l domain.Link
		var tagsJSON []byte
		var deletedAt sql.NullTime
		if err := rows.Scan(&l.ID, &l.OriginalURL, &l.ShortCode, &l.Title, &tagsJSON, &l.CreatedAt, &l.UpdatedAt, &deletedAt); err != nil {
			return nil, err
		}
		if deletedAt.Valid {
			l.DeletedAt = &deletedAt.Time
		}
		_ = json.Unmarshal(tagsJSON, &l.Tags)
		links = append(links, l)
	}
	return links, nil
}

func (r *SQLiteRepository) RecordVisit(ctx context.Context, visit *domain.Visit) error {
	query := `INSERT INTO visits (link_id, referer, user_agent, ip_hash, created_at) VALUES (?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, visit.LinkID, visit.Referer, visit.UserAgent, visit.IPHash, visit.CreatedAt.Format("2006-01-02 15:04:05"))
	return err
}

func (r *SQLiteRepository) GetLinkStats(ctx context.Context, linkID int64) (*domain.LinkStats, error) {
	stats := &domain.LinkStats{
		Referrers:   make(map[string]int64),
		DailyClicks: []domain.DailyClick{},
	}

	// Total Clicks
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM visits WHERE link_id = ?`, linkID).Scan(&stats.TotalClicks)
	if err != nil {
		return nil, err
	}

	// Referrers
	rows, err := r.db.QueryContext(ctx, `SELECT referer, COUNT(*) as c FROM visits WHERE link_id = ? GROUP BY referer ORDER BY c DESC LIMIT 10`, linkID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var ref string
		var count int64
		if err := rows.Scan(&ref, &count); err != nil {
			return nil, err
		}
		if ref == "" {
			ref = "Direct"
		}
		stats.Referrers[ref] = count
	}
	rows.Close()

	// Daily Clicks (Last 30 days)
	// SQLite date formatting: strftime('%Y-%m-%d', created_at)
	rows2, err := r.db.QueryContext(ctx, `
		SELECT strftime('%Y-%m-%d', created_at) as date, COUNT(*) 
		FROM visits 
		WHERE link_id = ? 
		GROUP BY date 
		ORDER BY date DESC 
		LIMIT 30`, linkID)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()
	for rows2.Next() {
		var dc domain.DailyClick
		if err := rows2.Scan(&dc.Date, &dc.Count); err != nil {
			return nil, err
		}
		stats.DailyClicks = append(stats.DailyClicks, dc)
	}

	return stats, nil
}

func (r *SQLiteRepository) GetDashboardStats(ctx context.Context, limit int, filters map[string]interface{}) ([]domain.Link, int64, error) {
	// 1. Get total system clicks (simple count)
	// Actually, the interface asks for []Link and int64 (which usually is total count of LINKS or CLICKS?).
	// The implementation plan says "Top 10 links by clicks, Total system clicks".
	// The return signature `([]domain.Link, int64, error)` in definitions.go matches ListLinks but here context is different.
	// Let's assume int64 is Total CLICKS for the dashboard summary.

	var totalSystemClicks int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM visits`).Scan(&totalSystemClicks)
	if err != nil {
		return nil, 0, err
	}

	// 2. Get Top Links by clicks with filters
	// We need to join visits or subquery.
	// "rank by most used" -> ORDER BY (SELECT COUNT(*) FROM visits WHERE visits.link_id = links.id) DESC

	query := `
		SELECT l.id, l.original_url, l.short_code, l.title, l.tags, l.created_at, l.updated_at,
		(SELECT COUNT(*) FROM visits v WHERE v.link_id = l.id) as click_count
		FROM links l
		WHERE l.deleted_at IS NULL
	`
	args := []interface{}{}

	if search, ok := filters["search"].(string); ok && search != "" {
		query += " AND (l.title LIKE ?)" // Requested filter by title
		args = append(args, "%"+search+"%")
	}

	if tag, ok := filters["tag"].(string); ok && tag != "" {
		query += " AND EXISTS (SELECT 1 FROM json_each(l.tags) WHERE value = ?)"
		args = append(args, tag)
	}

	if domainFilter, ok := filters["domain"].(string); ok && domainFilter != "" {
		// Simple string match for domain in original url
		query += " AND l.original_url LIKE ?"
		args = append(args, "%"+domainFilter+"%")
	}

	query += " ORDER BY click_count DESC LIMIT ?"
	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var links []domain.Link
	for rows.Next() {
		var l domain.Link
		var tagsJSON []byte
		var clickCount int64 // We don't have a field in Link for this yet, but we can ignore it or add it to domain if needed.
		// For now scanning into existing struct, ignoring the extra column? No, Scan must match columns.
		// We should probably add `Clicks` to domain.Link or just return it.
		// The interface returns `[]domain.Link`.
		// Strategy: Scan click_count into a dummy var for sorting, but we might want to display it.
		// The user wants "ranking", so showing keys is important.
		// I will modify domain.Link to add `Clicks` field (omitted if empty/0 presumably, or always valid). Easiest way.

		if err := rows.Scan(&l.ID, &l.OriginalURL, &l.ShortCode, &l.Title, &tagsJSON, &l.CreatedAt, &l.UpdatedAt, &clickCount); err != nil {
			return nil, 0, err
		}
		_ = json.Unmarshal(tagsJSON, &l.Tags)
		l.Clicks = clickCount
		links = append(links, l)
	}

	return links, totalSystemClicks, nil
}

// Ensure interface compliance
var _ ports.LinkRepository = (*SQLiteRepository)(nil)
