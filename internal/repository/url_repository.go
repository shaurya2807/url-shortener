package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shaurya2807/url-shortener/internal/model"
)

// SQL to create the urls table:
//
//	CREATE TABLE IF NOT EXISTS urls (
//	    id          BIGSERIAL PRIMARY KEY,
//	    original_url TEXT      NOT NULL,
//	    short_code  VARCHAR(10) NOT NULL UNIQUE,
//	    click_count BIGINT      NOT NULL DEFAULT 0,
//	    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
//	);
//	CREATE INDEX IF NOT EXISTS idx_urls_short_code ON urls (short_code);

type URLRepository struct {
	db *pgxpool.Pool
}

func NewURLRepository(db *pgxpool.Pool) *URLRepository {
	return &URLRepository{db: db}
}

func (r *URLRepository) Create(ctx context.Context, url *model.URL) error {
	const q = `
		INSERT INTO urls (original_url, short_code)
		VALUES ($1, $2)
		RETURNING id, click_count, created_at`

	return r.db.QueryRow(ctx, q, url.OriginalURL, url.ShortCode).
		Scan(&url.ID, &url.ClickCount, &url.CreatedAt)
}

func (r *URLRepository) ExistsShortCode(ctx context.Context, code string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM urls WHERE short_code = $1)`, code).
		Scan(&exists)
	return exists, err
}

func (r *URLRepository) GetByShortCode(ctx context.Context, code string) (*model.URL, error) {
	url := &model.URL{}
	err := r.db.QueryRow(ctx,
		`SELECT id, original_url, short_code, click_count, created_at FROM urls WHERE short_code = $1`, code).
		Scan(&url.ID, &url.OriginalURL, &url.ShortCode, &url.ClickCount, &url.CreatedAt)
	if err != nil {
		return nil, err
	}
	return url, nil
}

func (r *URLRepository) GetStats(ctx context.Context, code string) (*model.URL, error) {
	url := &model.URL{}
	err := r.db.QueryRow(ctx,
		`SELECT id, original_url, short_code, click_count, created_at FROM urls WHERE short_code = $1`, code).
		Scan(&url.ID, &url.OriginalURL, &url.ShortCode, &url.ClickCount, &url.CreatedAt)
	if err != nil {
		return nil, err
	}
	return url, nil
}

func (r *URLRepository) IncrementClickCount(ctx context.Context, code string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE urls SET click_count = click_count + 1 WHERE short_code = $1`, code)
	return err
}
