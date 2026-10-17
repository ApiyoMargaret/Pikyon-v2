-- migrations/000009_create_notifications.down.sql

DROP POLICY IF EXISTS notifications_owner_all ON notifications;
DROP TABLE IF EXISTS notifications CASCADE;
DROP TYPE IF EXISTS notification_type;