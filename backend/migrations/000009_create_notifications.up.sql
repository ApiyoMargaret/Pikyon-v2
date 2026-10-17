-- migrations/000009_create_notifications.up.sql

CREATE TYPE notification_type AS ENUM (
    'memory_viewed',
    'memory_reacted',
    'storage_warning',
    'share_accepted',
    'system'
);

CREATE TABLE notifications (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type        notification_type NOT NULL,
    title       TEXT NOT NULL,
    message     TEXT NOT NULL,
    metadata    JSONB,
    is_read     BOOLEAN NOT NULL DEFAULT FALSE,
    read_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Optimized partial index for unread notification feeds
CREATE INDEX idx_notifications_unread 
    ON notifications(user_id, created_at DESC) 
    WHERE is_read = FALSE;

-- Row Level Security
ALTER TABLE notifications ENABLE ROW LEVEL SECURITY;

CREATE POLICY notifications_owner_all ON notifications 
    FOR ALL 
    USING (auth.uid() = user_id)
    WITH CHECK (auth.uid() = user_id);