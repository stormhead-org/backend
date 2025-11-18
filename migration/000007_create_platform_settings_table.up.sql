CREATE TABLE IF NOT EXISTS platform_settings (
    id SERIAL PRIMARY KEY,
    platform_owner_id UUID,
    FOREIGN KEY (platform_owner_id) REFERENCES users(id) ON DELETE SET NULL
);

INSERT INTO platform_settings (id) VALUES (1);
