CREATE TABLE IF NOT EXISTS "user" (
    id UUID PRIMARY KEY,
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    password TEXT NOT NULL,
    salt TEXT NOT NULL,
    verification_token TEXT NOT NULL,
    reset_token TEXT NOT NULL,
    is_verified BOOLEAN,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS "session" (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    user_agent TEXT NOT NULL,
    ip_address TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS "community" (
    id UUID PRIMARY KEY,
    owner_id UUID NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    rules TEXT NOT NULL,
    is_banned BOOLEAN,
    ban_reason TEXT,
    member_count INTEGER DEFAULT 0,
    post_count INTEGER DEFAULT 0,
    reputation INTEGER DEFAULT 0,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS "community_user" (
    community_id UUID NOT NULL,
    user_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS "post" (
    id UUID PRIMARY KEY,
    community_id UUID NOT NULL,
    author_id UUID NOT NULL,
    title TEXT NOT NULL,
    content JSONB NOT NULL,
    status INTEGER,
    like_count INTEGER,
    comment_count INTEGER,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    published_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS "post_like" (
    id UUID PRIMARY KEY,
    post_id UUID NOT NULL,
    user_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS "bookmark" (
    id UUID PRIMARY KEY,
    post_id UUID NOT NULL,
    user_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS "comment" (
    id UUID PRIMARY KEY,
    parent_comment_id UUID,
    post_id UUID NOT NULL,
    author_id UUID NOT NULL,
    content TEXT NOT NULL,
    like_count INTEGER,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS "comment_like" (
    id UUID PRIMARY KEY,
    comment_id UUID NOT NULL,
    user_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS "follower" (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    follower_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
