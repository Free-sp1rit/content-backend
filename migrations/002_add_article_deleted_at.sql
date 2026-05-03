ALTER TABLE articles
ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ NULL;

CREATE INDEX IF NOT EXISTS idx_articles_not_deleted_state_created_at
    ON articles (state, created_at DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_articles_not_deleted_author_id_created_at
    ON articles (author_id, created_at DESC)
    WHERE deleted_at IS NULL;
