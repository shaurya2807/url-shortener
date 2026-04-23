package service

import (
	"context"
	"crypto/rand"
	"fmt"

	"github.com/shaurya2807/url-shortener/internal/model"
	"github.com/shaurya2807/url-shortener/internal/repository"
	"go.uber.org/zap"
)

const (
	shortCodeLen = 6
	charset      = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

type URLService struct {
	repo    *repository.URLRepository
	baseURL string
	log     *zap.Logger
}

func NewURLService(repo *repository.URLRepository, baseURL string, log *zap.Logger) *URLService {
	return &URLService{repo: repo, baseURL: baseURL, log: log}
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
