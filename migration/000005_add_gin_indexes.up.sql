CREATE INDEX idx_posts_fts ON post USING gin (to_tsvector('english', title || ' ' || content::text));
CREATE INDEX idx_comments_fts ON comment USING gin (to_tsvector('english', content));
