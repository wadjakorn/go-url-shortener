package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	_ "github.com/tursodatabase/libsql-client-go/libsql" // Turso driver
	"github.com/wadjakorntonsri/go-url-shortener/pkg/core/domain"
	"github.com/wadjakorntonsri/go-url-shortener/pkg/ports"
	_ "modernc.org/sqlite" // Local SQLite driver
)

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(dbURL string) (*SQLiteRepository, error) {
	driverName := "sqlite"
	if strings.Contains(dbURL, "libsql://") || strings.Contains(dbURL, "wss://") {
		driverName = "libsql"
	}

	db, err := sql.Open(driverName, dbURL)
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
		clicks INTEGER DEFAULT 0,
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

	CREATE TABLE IF NOT EXISTS collections (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		slug TEXT NOT NULL UNIQUE,
		title TEXT,
		description TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_collections_slug ON collections(slug);

	CREATE TABLE IF NOT EXISTS collection_links (
		collection_id INTEGER NOT NULL,
		link_id INTEGER NOT NULL,
		sort_order INTEGER DEFAULT 0,
		PRIMARY KEY (collection_id, link_id),
		FOREIGN KEY(collection_id) REFERENCES collections(id) ON DELETE CASCADE,
		FOREIGN KEY(link_id) REFERENCES links(id) ON DELETE CASCADE
	);
	`
	if _, err := db.Exec(query); err != nil {
		return err
	}

	// Migration for existing tables: Add 'clicks' column if it doesn't exist
	// SQLite doesn't support IF NOT EXISTS for ADD COLUMN, so we just try and ignore error
	// or check pragma table_info. Simpler is just to try.
	_, _ = db.Exec(`ALTER TABLE links ADD COLUMN clicks INTEGER DEFAULT 0`)

	return nil
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
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Insert Visit Record
	queryVisit := `INSERT INTO visits (link_id, referer, user_agent, ip_hash, created_at) VALUES (?, ?, ?, ?, ?)`
	_, err = tx.ExecContext(ctx, queryVisit, visit.LinkID, visit.Referer, visit.UserAgent, visit.IPHash, visit.CreatedAt.Format("2006-01-02 15:04:05"))
	if err != nil {
		return err
	}

	// 2. Increment Link Clicks Counter (Atomic)
	queryCount := `UPDATE links SET clicks = clicks + 1 WHERE id = ?`
	_, err = tx.ExecContext(ctx, queryCount, visit.LinkID)
	if err != nil {
		return err
	}

	return tx.Commit()
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
	// 1. Get total system clicks (Optimized)
	// Instead of scanning visits, we can sum the clicks column. Much faster.
	var totalSystemClicks int64
	err := r.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(clicks), 0) FROM links WHERE deleted_at IS NULL`).Scan(&totalSystemClicks)
	if err != nil {
		return nil, 0, err
	}

	// 2. Get Top Links by clicks (Optimized)
	// Uses the indexed 'clicks' column (we should add an index later if needed, but for now it's a simple sort)
	query := `
		SELECT id, original_url, short_code, title, tags, clicks, created_at, updated_at
		FROM links
		WHERE deleted_at IS NULL
	`
	args := []interface{}{}

	if search, ok := filters["search"].(string); ok && search != "" {
		query += " AND (title LIKE ?)"
		args = append(args, "%"+search+"%")
	}

	if tag, ok := filters["tag"].(string); ok && tag != "" {
		query += " AND EXISTS (SELECT 1 FROM json_each(tags) WHERE value = ?)"
		args = append(args, tag)
	}

	if domainFilter, ok := filters["domain"].(string); ok && domainFilter != "" {
		query += " AND original_url LIKE ?"
		args = append(args, "%"+domainFilter+"%")
	}

	query += " ORDER BY clicks DESC LIMIT ?"
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

		if err := rows.Scan(&l.ID, &l.OriginalURL, &l.ShortCode, &l.Title, &tagsJSON, &l.Clicks, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, 0, err
		}
		_ = json.Unmarshal(tagsJSON, &l.Tags)
		links = append(links, l)
	}

	return links, totalSystemClicks, nil
}

// --- Collection Repository Implementation ---

func (r *SQLiteRepository) CreateCollection(ctx context.Context, collection *domain.Collection) error {
	query := `INSERT INTO collections (slug, title, description, created_at, updated_at) 
			  VALUES (?, ?, ?, ?, ?) RETURNING id`

	res, err := r.db.ExecContext(ctx, query, collection.Slug, collection.Title, collection.Description, collection.CreatedAt, collection.UpdatedAt)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	collection.ID = id
	return nil
}

func (r *SQLiteRepository) GetCollection(ctx context.Context, id int64) (*domain.Collection, error) {
	query := `SELECT id, slug, title, description, created_at, updated_at FROM collections WHERE id = ?`
	row := r.db.QueryRowContext(ctx, query, id)

	var c domain.Collection
	if err := row.Scan(&c.ID, &c.Slug, &c.Title, &c.Description, &c.CreatedAt, &c.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *SQLiteRepository) GetCollectionBySlug(ctx context.Context, slug string) (*domain.Collection, error) {
	query := `SELECT id, slug, title, description, created_at, updated_at FROM collections WHERE slug = ?`
	row := r.db.QueryRowContext(ctx, query, slug)

	var c domain.Collection
	if err := row.Scan(&c.ID, &c.Slug, &c.Title, &c.Description, &c.CreatedAt, &c.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *SQLiteRepository) UpdateCollection(ctx context.Context, collection *domain.Collection) error {
	query := `UPDATE collections SET slug = ?, title = ?, description = ?, updated_at = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, collection.Slug, collection.Title, collection.Description, collection.UpdatedAt, collection.ID)
	return err
}

func (r *SQLiteRepository) DeleteCollection(ctx context.Context, id int64) error {
	query := `DELETE FROM collections WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *SQLiteRepository) ListCollections(ctx context.Context, limit, offset int, filters map[string]interface{}) ([]domain.Collection, error) {
	query := `SELECT id, slug, title, description, created_at, updated_at FROM collections`
	args := []interface{}{}

	if search, ok := filters["search"].(string); ok && search != "" {
		query += " WHERE title LIKE ? OR slug LIKE ?"
		args = append(args, "%"+search+"%", "%"+search+"%")
	}

	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var collections []domain.Collection
	for rows.Next() {
		var c domain.Collection
		if err := rows.Scan(&c.ID, &c.Slug, &c.Title, &c.Description, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		collections = append(collections, c)
	}
	return collections, nil
}

func (r *SQLiteRepository) AddLinkToCollection(ctx context.Context, collectionID, linkID int64) error {
	query := `INSERT INTO collection_links (collection_id, link_id) VALUES (?, ?)`
	_, err := r.db.ExecContext(ctx, query, collectionID, linkID)
	return err
}

func (r *SQLiteRepository) RemoveLinkFromCollection(ctx context.Context, collectionID, linkID int64) error {
	query := `DELETE FROM collection_links WHERE collection_id = ? AND link_id = ?`
	_, err := r.db.ExecContext(ctx, query, collectionID, linkID)
	return err
}

func (r *SQLiteRepository) UpdateLinkOrder(ctx context.Context, collectionID, linkID int64, newOrder int) error {
	query := `UPDATE collection_links SET sort_order = ? WHERE collection_id = ? AND link_id = ?`
	_, err := r.db.ExecContext(ctx, query, newOrder, collectionID, linkID)
	return err
}

func (r *SQLiteRepository) GetCollectionLinks(ctx context.Context, collectionID int64) ([]domain.Link, error) {
	query := `SELECT l.id, l.original_url, l.short_code, l.title, l.tags, l.created_at, l.updated_at
			  FROM links l
			  JOIN collection_links cl ON l.id = cl.link_id
			  WHERE cl.collection_id = ? AND l.deleted_at IS NULL
			  ORDER BY cl.sort_order ASC`

	rows, err := r.db.QueryContext(ctx, query, collectionID)
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

// Ensure interface compliance
var _ ports.LinkRepository = (*SQLiteRepository)(nil)
