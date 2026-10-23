-- 000010_create_notifications.up.sql
CREATE TYPE notification_type AS ENUM (
    'memory_viewed', 'memory_reacted', 'storage_warning', 'share_accepted', 'system'
);

CREATE TABLE notifications (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type        notification_type NOT NULL,
    title       TEXT NOT NULL,
    message     TEXT NOT NULL,
    metadata    JSONB,
    is_read     BOOLEAN NOT NULL DEFAULT FALSE,
    read_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notifications_unread ON notifications(user_id, created_at DESC) WHERE is_read = FALSE;
CREATE INDEX idx_notifications_user_id ON notifications(user_id);

ALTER TABLE notifications ENABLE ROW LEVEL SECURITY;

CREATE POLICY notifications_owner_all ON notifications FOR ALL TO pikyon_app
    USING (user_id = (SELECT pikyon.current_user_id()))
    WITH CHECK (user_id = (SELECT pikyon.current_user_id()));

-- Notifications are created on behalf of OTHER users (e.g. "your memory was viewed"),
-- so INSERT happens on the privileged connection only. Users read, mark read, and delete.
GRANT SELECT, DELETE ON notifications TO pikyon_app;
GRANT UPDATE (is_read, read_at) ON notifications TO pikyon_app;