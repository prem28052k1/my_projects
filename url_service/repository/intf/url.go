package repository

import (
	"context"

	"github.com/my_projects/url_service/model"
)

type Url interface {
	Save(ctx context.Context, url *model.Url) error
	GetByUrl(ctx context.Context, originalUrl string) (*model.Url, error)
	GetByShortUrl(ctx context.Context, shortUrl string) (*model.Url, error)
	UpdateClickCount(ctx context.Context, shortUrl string) error
	List(ctx context.Context, offset, limit int) ([]*model.Url, int64, error)
}
