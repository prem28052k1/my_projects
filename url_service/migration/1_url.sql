CREATE TABLE IF NOT EXISTS url (
    url_id VARCHAR(50) PRIMARY KEY,
    url TEXT NOT NULL,
    short_url VARCHAR(10) UNIQUE NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    click_count BIGINT NOT NULL DEFAULT 0,
    last_accessed_at TIMESTAMP
);

-- Index for looking up by original URL (for idempotency)
CREATE INDEX IF NOT EXISTS idx_url_original ON url(url);

-- Index for looking up by short_url (most common operation)
CREATE INDEX IF NOT EXISTS idx_url_short ON url(short_url);

-- Index for sorting by created_at (for list operations)
CREATE INDEX IF NOT EXISTS idx_url_created_at ON url(created_at DESC);