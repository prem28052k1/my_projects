package model

import "time"

type Url struct {
	Url            string    `json:"url"`
	UrlId          string    `json:"url_id"`
	ShortUrl       string    `json:"short_url"`
	CreatedAt      time.Time `json:"created"`
	ClickCount     int64     `json:"click_count"`
	LastAccessedAt time.Time `json:"last_accessed_at"`
}
