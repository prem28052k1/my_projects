package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/my_projects/url_service/gen"
	"github.com/my_projects/url_service/model"
	repository "github.com/my_projects/url_service/repository/intf"
	"github.com/my_projects/url_service/util"
)

type URLServiceImpl struct {
	gen.UnimplementedUrlServiceServer
	repo repository.Url
}

func NewURLServiceImpl(repo repository.Url) *URLServiceImpl {
	return &URLServiceImpl{
		repo: repo,
	}
}

func (u *URLServiceImpl) Shorten(ctx context.Context, req *gen.ShortenUrlRequest) (*gen.ShortenUrlResponse, error) {
	if err := util.ValidateURL(req.Url); err != nil {
		slog.Error("invalid URL provided", "err", err, "url", req.Url)
		return nil, err
	}

	existingUrl, err := u.repo.GetByUrl(ctx, req.Url)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		slog.Error("error while fetching existing url", "err", err, "url", req.Url)
		return nil, errors.New("failed to check existing URL")
	}

	// if url already exists return
	if existingUrl != nil {
		return &gen.ShortenUrlResponse{
			UrlId:    existingUrl.UrlId,
			ShortUrl: existingUrl.ShortUrl,
		}, nil
	}

	// Generate short code from URL (max 10 characters)
	shortCode := generateShortCode(req.Url)
	urlId := fmt.Sprintf("url_%d", time.Now().UnixNano())

	// Create URL model
	urlModel := &model.Url{
		Url:            req.Url,
		UrlId:          urlId,
		ShortUrl:       shortCode,
		CreatedAt:      time.Now(),
		LastAccessedAt: time.Time{},
	}

	// Save to repository
	err = u.repo.Save(ctx, urlModel)
	if err != nil {
		slog.Error("error while inserting url", "err", err, "url_id", urlId)
		return nil, errors.New("failed to save URL")
	}

	return &gen.ShortenUrlResponse{
		UrlId:    urlId,
		ShortUrl: shortCode,
	}, nil
}

// Expand retrieves the original URL from a short URL and increments click count
func (u *URLServiceImpl) Expand(ctx context.Context, req *gen.ExpandUrlRequest) (*gen.ExpandUrlResponse, error) {
	if req.ShortUrl == "" {
		slog.Error("empty short URL provided")
		return nil, errors.New("short URL cannot be empty")
	}

	// Get URL by short code
	urlData, err := u.repo.GetByShortUrl(ctx, req.ShortUrl)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			slog.Error("short URL not found", "short_url", req.ShortUrl)
			return nil, errors.New("short URL not found")
		}
		slog.Error("error while fetching URL by short code", "err", err, "short_url", req.ShortUrl)
		return nil, errors.New("failed to fetch URL")
	}

	// Update click count asynchronously (fire and forget to keep latency low)
	go func() {
		if err := u.repo.UpdateClickCount(context.Background(), req.ShortUrl); err != nil {
			slog.Error("failed to update click count", "err", err, "short_url", req.ShortUrl)
		}
	}()

	return &gen.ExpandUrlResponse{
		OriginalUrl: urlData.Url,
		ClickCount:  urlData.ClickCount,
		CreatedAt:   urlData.CreatedAt.Format(time.RFC3339),
	}, nil
}

// ListUrls returns a paginated list of all shortened URLs (admin function)
func (u *URLServiceImpl) ListUrls(ctx context.Context, req *gen.ListUrlsRequest) (*gen.ListUrlsResponse, error) {
	// Set default pagination values
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100 
	}

	offset := int((page - 1) * pageSize)
	limit := int(pageSize)

	// Get paginated URLs
	urls, totalCount, err := u.repo.List(ctx, offset, limit)
	if err != nil {
		slog.Error("error while listing URLs", "err", err, "page", page, "page_size", pageSize)
		return nil, errors.New("failed to list URLs")
	}

	// Convert to response format
	var urlInfos []*gen.UrlInfo
	for _, url := range urls {
		lastAccessed := ""
		if !url.LastAccessedAt.IsZero() {
			lastAccessed = url.LastAccessedAt.Format(time.RFC3339)
		}

		urlInfos = append(urlInfos, &gen.UrlInfo{
			UrlId:          url.UrlId,
			OriginalUrl:    url.Url,
			ShortUrl:       url.ShortUrl,
			ClickCount:     url.ClickCount,
			CreatedAt:      url.CreatedAt.Format(time.RFC3339),
			LastAccessedAt: lastAccessed,
		})
	}

	return &gen.ListUrlsResponse{
		Urls:       urlInfos,
		TotalCount: totalCount,
		Page:       page,
		PageSize:   pageSize,
	}, nil
}

// generateShortCode creates a unique short code from a URL (max 10 characters)
func generateShortCode(url string) string {
	hash := sha256.Sum256([]byte(url))
	// Use base64 URL encoding to get URL-safe characters
	encoded := base64.RawURLEncoding.EncodeToString(hash[:])
	// Take first 10 characters to keep it short
	if len(encoded) > 10 {
		return encoded[:10]
	}
	return encoded
}
