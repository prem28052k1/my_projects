package pgx

import (
	"context"

	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/my_projects/url_service/model"
	repository "github.com/my_projects/url_service/repository/intf"
)

// UrlRepository implements the repository.Url interface using pgx
type UrlRepository struct {
	pool *pgxpool.Pool
}

// NewUrlRepository creates a new instance of UrlRepository
func NewUrlRepository(pool *pgxpool.Pool) repository.Url {
	return &UrlRepository{
		pool: pool,
	}
}

// Save inserts a new URL record into the database using a transaction
func (r *UrlRepository) Save(ctx context.Context, url *model.Url) error {
	// Begin transaction
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		slog.Error("error while beginning transaction", "err", err, "url_id", url.UrlId)
		return err
	}

	// Defer rollback in case of error
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	// Execute insert query
	query := `
		INSERT INTO url (url, url_id, short_url, created_at, click_count, last_accessed_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err = tx.Exec(ctx, query,
		url.Url,
		url.UrlId,
		url.ShortUrl,
		url.CreatedAt,
		url.ClickCount,
		url.LastAccessedAt,
	)
	if err != nil {
		slog.Error("error while executing insert query", "err", err, "url_id", url.UrlId)
		return err
	}

	// Commit transaction
	if err = tx.Commit(ctx); err != nil {
		slog.Error("error while committing transaction", "err", err, "url_id", url.UrlId)
		return err
	}

	return nil
}

// GetByUrl finds a URL record by its original URL
func (r *UrlRepository) GetByUrl(ctx context.Context, originalUrl string) (*model.Url, error) {
	query := `
		SELECT url, url_id, short_url, created_at, click_count, last_accessed_at
		FROM url
		WHERE url = $1
		LIMIT 1
	`

	var url model.Url
	err := r.pool.QueryRow(ctx, query, originalUrl).Scan(
		&url.Url,
		&url.UrlId,
		&url.ShortUrl,
		&url.CreatedAt,
		&url.ClickCount,
		&url.LastAccessedAt,
	)
	if err != nil {
		slog.Error("error while finding url by original url", "err", err, "url", originalUrl)
		return nil, err
	}

	return &url, nil
}

// GetByShortUrl finds a URL record by its short URL
func (r *UrlRepository) GetByShortUrl(ctx context.Context, shortUrl string) (*model.Url, error) {
	query := `
		SELECT url, url_id, short_url, created_at, click_count, last_accessed_at
		FROM url
		WHERE short_url = $1
		LIMIT 1
	`

	var url model.Url
	err := r.pool.QueryRow(ctx, query, shortUrl).Scan(
		&url.Url,
		&url.UrlId,
		&url.ShortUrl,
		&url.CreatedAt,
		&url.ClickCount,
		&url.LastAccessedAt,
	)
	if err != nil {
		slog.Error("error while finding url by short url", "err", err, "short_url", shortUrl)
		return nil, err
	}

	return &url, nil
}

// UpdateClickCount increments the click count and updates last_accessed_at
func (r *UrlRepository) UpdateClickCount(ctx context.Context, shortUrl string) error {
	query := `
		UPDATE url
		SET click_count = click_count + 1,
		    last_accessed_at = NOW()
		WHERE short_url = $1
	`

	result, err := r.pool.Exec(ctx, query, shortUrl)
	if err != nil {
		slog.Error("error while updating click count", "err", err, "short_url", shortUrl)
		return err
	}

	if result.RowsAffected() == 0 {
		slog.Error("no rows updated for click count", "short_url", shortUrl)
		return nil // Not an error, just no matching URL
	}

	return nil
}

// List returns paginated list of URLs with total count
func (r *UrlRepository) List(ctx context.Context, offset, limit int) ([]*model.Url, int64, error) {
	// Get total count
	var totalCount int64
	countQuery := `SELECT COUNT(*) FROM url`
	err := r.pool.QueryRow(ctx, countQuery).Scan(&totalCount)
	if err != nil {
		slog.Error("error while counting urls", "err", err)
		return nil, 0, err
	}

	// Get paginated results
	query := `
		SELECT url, url_id, short_url, created_at, click_count, last_accessed_at
		FROM url
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		slog.Error("error while listing urls", "err", err, "offset", offset, "limit", limit)
		return nil, 0, err
	}
	defer rows.Close()

	var urls []*model.Url
	for rows.Next() {
		var url model.Url
		err := rows.Scan(
			&url.Url,
			&url.UrlId,
			&url.ShortUrl,
			&url.CreatedAt,
			&url.ClickCount,
			&url.LastAccessedAt,
		)
		if err != nil {
			slog.Error("error while scanning url row", "err", err)
			return nil, 0, err
		}
		urls = append(urls, &url)
	}

	if err = rows.Err(); err != nil {
		slog.Error("error iterating url rows", "err", err)
		return nil, 0, err
	}

	return urls, totalCount, nil
}
