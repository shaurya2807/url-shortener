package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/shaurya2807/url-shortener/internal/cache"
	"github.com/shaurya2807/url-shortener/internal/model"
	"github.com/shaurya2807/url-shortener/internal/repository"
	"go.uber.org/zap"
)

var ErrNotFound = errors.New("short code not found")

const (
	shortCodeLen = 6
	charset      = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

type URLService struct {
	repo    *repository.URLRepository
	cache   *cache.Cache
	baseURL string
	log     *zap.Logger
}

func NewURLService(repo *repository.URLRepository, cache *cache.Cache, baseURL string, log *zap.Logger) *URLService {
	return &URLService{repo: repo, cache: cache, baseURL: baseURL, log: log}
}

func (s *URLService) Shorten(ctx context.Context, originalURL string) (*model.ShortenResponse, error) {
	code, err := s.generateUniqueCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("generate short code: %w", err)
	}

	url := &model.URL{OriginalURL: originalURL, ShortCode: code}
	if err := s.repo.Create(ctx, url); err != nil {
		return nil, fmt.Errorf("store url: %w", err)
	}

	s.log.Info("url shortened",
		zap.String("short_code", code),
		zap.String("original_url", originalURL),
	)

	return &model.ShortenResponse{
		ShortCode:   code,
		ShortURL:    s.baseURL + "/" + code,
		OriginalURL: originalURL,
		CreatedAt:   url.CreatedAt,
	}, nil
}

func (s *URLService) Redirect(ctx context.Context, code string) (string, error) {
	if cached, err := s.cache.Get(ctx, code); err == nil && cached != "" {
		s.log.Info("cache hit", zap.String("short_code", code))
		return cached, nil
	}

	url, err := s.repo.GetByShortCode(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("get by short code: %w", err)
	}

	s.log.Info("cache miss", zap.String("short_code", code))

	go func() {
		if err := s.cache.Set(context.Background(), code, url.OriginalURL); err != nil {
			s.log.Error("cache set failed", zap.String("short_code", code), zap.Error(err))
		}
	}()

	go func() {
		if err := s.repo.IncrementClickCount(context.Background(), code); err != nil {
			s.log.Error("increment click count failed", zap.String("short_code", code), zap.Error(err))
		}
	}()

	s.log.Info("redirect",
		zap.String("short_code", code),
		zap.String("original_url", url.OriginalURL),
	)

	return url.OriginalURL, nil
}

func (s *URLService) GetStats(ctx context.Context, code string) (*model.StatsResponse, error) {
	url, err := s.repo.GetStats(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get stats: %w", err)
	}
	return &model.StatsResponse{
		ShortCode:   url.ShortCode,
		ShortURL:    s.baseURL + "/" + url.ShortCode,
		OriginalURL: url.OriginalURL,
		ClickCount:  url.ClickCount,
		CreatedAt:   url.CreatedAt,
	}, nil
}

func (s *URLService) generateUniqueCode(ctx context.Context) (string, error) {
	for {
		code, err := randomCode()
		if err != nil {
			return "", err
		}
		exists, err := s.repo.ExistsShortCode(ctx, code)
		if err != nil {
			return "", err
		}
		if !exists {
			return code, nil
		}
	}
}

func randomCode() (string, error) {
	b := make([]byte, shortCodeLen)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("crypto/rand: %w", err)
	}
	for i, v := range b {
		b[i] = charset[int(v)%len(charset)]
	}
	return string(b), nil
}
