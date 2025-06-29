-- Remove 'invited' status from group_members table
-- First, delete any records with 'invited' status
DELETE FROM group_members WHERE status = 'invited';

-- Create a new table with the original constraint
CREATE TABLE group_members_new (
    id TEXT PRIMARY KEY,
    group_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('creator', 'admin', 'member')),
    status TEXT NOT NULL CHECK (status IN ('pending', 'accepted', 'rejected')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE (group_id, user_id)
);

-- Copy data from the old table to the new table
INSERT INTO group_members_new (id, group_id, user_id, role, status, created_at, updated_at)
SELECT id, group_id, user_id, role, status, created_at, updated_at
FROM group_members;

-- Drop the old table
DROP TABLE group_members;

-- Rename the new table to the original name
ALTER TABLE group_members_new RENAME TO group_members;
