package util

import (
	"errors"
	"net/url"
	"strings"
)

// ValidateURL checks if the provided string is a valid URL
func ValidateURL(urlStr string) error {
	if urlStr == "" {
		return errors.New("URL cannot be empty")
	}

	// Parse the URL
	parsedURL, err := url.ParseRequestURI(urlStr)
	if err != nil {
		return errors.New("invalid URL format")
	}

	// Check scheme (must be http or https)
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return errors.New("URL must use http or https scheme")
	}

	// Check if host is present
	if parsedURL.Host == "" {
		return errors.New("URL must have a valid host")
	}

	// Check URL length (max 2048 characters as per RFC)
	if len(urlStr) > 2048 {
		return errors.New("URL exceeds maximum length of 2048 characters")
	}

	// Ensure it's not just the scheme
	if strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(urlStr, "https://"), "http://")) == "" {
		return errors.New("URL must contain more than just the scheme")
	}

	return nil
}
