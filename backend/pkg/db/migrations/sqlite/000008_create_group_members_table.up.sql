CREATE TABLE IF NOT EXISTS group_members (
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
