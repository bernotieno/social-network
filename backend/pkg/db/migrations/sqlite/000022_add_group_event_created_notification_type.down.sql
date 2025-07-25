-- Remove group_event_created notification type
-- First, create a new table with the original constraint
CREATE TABLE notifications_new (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    sender_id TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('follow_request', 'follow_accepted', 'new_follower', 'post_like', 'post_comment', 'group_invite', 'group_join_request', 'group_join_approved', 'group_join_rejected', 'event_invite')),
    content TEXT NOT NULL,
    data TEXT,
    read_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (sender_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Copy data from the old table to the new table (excluding group_event_created notifications)
INSERT INTO notifications_new (id, user_id, sender_id, type, content, data, read_at, created_at)
SELECT id, user_id, sender_id, type, content, data, read_at, created_at
FROM notifications
WHERE type != 'group_event_created';

-- Drop the old table
DROP TABLE notifications;

-- Rename the new table to the original name
ALTER TABLE notifications_new RENAME TO notifications;
